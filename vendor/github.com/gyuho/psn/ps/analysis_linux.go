package ps

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
)

// Table represents CSV tables.
type Table struct {
	// Columns maps its column name to its column index.
	Columns     map[string]int
	ColumnSlice []string
	Rows        [][]string

	MinTS int64
	MaxTS int64
}

var (
	ColumnsPS = make(map[string]int)
)

func init() {
	for i, v := range append([]string{"unix_ts"}, ProcessTableColumns...) {
		ColumnsPS[v] = i
	}
}

// ReadCSV reads a csv file for ps csv file.
// It assumes the results from one single program.
func ReadCSV(columns map[string]int, fpath string) (Table, error) {
	f, err := openToRead(fpath)
	if err != nil {
		return Table{}, err
	}
	defer f.Close()

	columnSlice := make([]string, len(columns))
	for k, v := range columns {
		columnSlice[v] = k
	}

	rd := csv.NewReader(f)

	// in case that rows have different number of fields
	rd.FieldsPerRecord = -1

	rows, err := rd.ReadAll()
	if err != nil {
		return Table{}, err
	}
	if len(rows) > 0 && len(rows[0]) > 0 {
		if rows[0][0] == columnSlice[0] {
			rows = rows[1:]
		}
	}
	min, err := strconv.ParseInt(rows[0][0], 10, 64)
	if err != nil {
		return Table{}, err
	}
	max, err := strconv.ParseInt(rows[len(rows)-1][0], 10, 64)
	if err != nil {
		return Table{}, err
	}
	return Table{
		Columns:     columns,
		ColumnSlice: columnSlice,
		Rows:        rows,
		MinTS:       min,
		MaxTS:       max,
	}, nil
}

// ReadCSVs reads multiple csv files, including only unix timestamps,
// CPU usage, and VmRSS in MB. It assumes the results from one single program.
func ReadCSVs(columns map[string]int, fpaths ...string) (Table, error) {
	tbs := []Table{}
	var (
		minTS int64 = math.MaxInt64
		maxTS int64 = -1
	)
	for _, fpath := range fpaths {
		tb, err := ReadCSV(columns, fpath)
		if err != nil {
			return Table{}, err
		}
		tbs = append(tbs, tb)

		if minTS > tb.MinTS {
			minTS = tb.MinTS
		}
		if maxTS < tb.MaxTS {
			maxTS = tb.MaxTS
		}
	}

	ntb := Table{}
	ntb.MinTS = minTS
	ntb.MaxTS = maxTS
	ntb.Columns = make(map[string]int)
	ntb.Columns["unix_tx"] = 0
	for i := range fpaths {
		ntb.Columns[fmt.Sprintf("cpu_%d", i+1)] = 2*i + 1
		ntb.Columns[fmt.Sprintf("memory_mb_%d", i+1)] = 2*i + 2
	}
	columnSlice := make([]string, len(ntb.Columns))
	for k, v := range ntb.Columns {
		columnSlice[v] = k
	}
	ntb.ColumnSlice = columnSlice

	cIdx, mIdx := tbs[0].Columns["CpuUsageFloat64"], tbs[0].Columns["VmRSSBytes"]

	nrows := [][]string{}
	for i, tb := range tbs {
		if minTS < tb.MinTS { // need to fill in top rows
			// push-front from minTS to tb.MinTS-1
			rows := make([][]string, tb.MinTS-minTS)
			for i := range rows {
				emptyRow := append([]string{fmt.Sprintf("%d", minTS+int64(i))}, strings.Split(strings.Repeat("0.00_", len(ProcessTableColumns)), "_")...)
				rows[i] = emptyRow
			}
			tb.Rows = append(rows, tb.Rows...)
		}
		if maxTS > tb.MaxTS { // need to fill in bottom rows
			// push-back from tb.MaxTS+1 to maxTS
			rows := make([][]string, maxTS-tb.MaxTS)
			for i := range rows {
				emptyRow := append([]string{fmt.Sprintf("%d", tb.MaxTS+int64(i))}, strings.Split(strings.Repeat("0.00_", len(ProcessTableColumns)), "_")...)
				rows[i] = emptyRow
			}
			tb.Rows = append(tb.Rows, rows...)
		}

		if i == 0 {
			for _, row := range tb.Rows {
				if row[0] == tb.ColumnSlice[0] {
					continue
				}
				mf, _ := strconv.ParseFloat(row[mIdx], 64)
				nrows = append(nrows, []string{row[0], row[cIdx], fmt.Sprintf("%.2f", mf/(1000*1000))}) // memory usage in mega-bytes
			}
		} else {
			for rowNum := range nrows {
				trow := tb.Rows[rowNum]
				if trow[0] == tb.ColumnSlice[0] {
					continue
				}
				mf, _ := strconv.ParseFloat(trow[mIdx], 64)
				nrows[rowNum] = append(nrows[rowNum], []string{trow[cIdx], fmt.Sprintf("%.2f", mf/(1000*1000))}...)
			}
		}
	}
	ntb.Rows = nrows

	return ntb, nil
}

// ToRows returns rows from the table.
func (t Table) ToRows() [][]string {
	return append([][]string{t.ColumnSlice}, t.Rows...)
}

// ToCSV saves the table to csv file.
func (t Table) ToCSV(fpath string) error {
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		f, err = os.Create(fpath)
		if err != nil {
			return err
		}
	}
	defer f.Close()

	wr := csv.NewWriter(f)

	if err := wr.Write(t.ColumnSlice); err != nil {
		return err
	}

	if err := wr.WriteAll(t.Rows); err != nil {
		return err
	}

	wr.Flush()
	return wr.Error()
}

func ReadCSVFillIn(fpath string) (Table, error) {
	tb, err := ReadCSV(ColumnsPS, fpath)
	if err != nil {
		return Table{}, err
	}
	var (
		uidx    = tb.Columns["unix_ts"]
		tsToRow = make(map[int64][]string)
	)
	for _, row := range tb.Rows {
		num, err := strconv.ParseInt(row[uidx], 10, 64)
		if err != nil {
			return Table{}, err
		}
		tsToRow[num] = row
	}

	var nrows [][]string
	for i := tb.MinTS; i < tb.MaxTS; i++ {
		r, ok := tsToRow[i]
		if !ok {
			row, err := getEstimate(i, tb.MinTS, tb.MaxTS, tsToRow)
			if err != nil {
				return tb, err
			}
			tsToRow[i] = row
			nrows = append(nrows, row)
			continue
		}
		nrows = append(nrows, r)
	}
	tb.Rows = nrows

	return tb, nil
}

// getEstimate estimates the row when ts does not exist in table.
func getEstimate(ts, min, max int64, tsToRow map[int64][]string) ([]string, error) {
	// get the nearest lower value
	var lower []string
	for i := ts - 1; i >= min; i-- {
		row, ok := tsToRow[i]
		if !ok {
			continue
		}
		lower = row
		break
	}

	// get the nearest upper value
	var upper []string
	for i := ts + 1; i <= max; i++ {
		row, ok := tsToRow[i]
		if !ok {
			continue
		}
		upper = row
		break
	}

	nrow := make([]string, len(lower))
	copy(nrow, lower)
	nrow[0] = fmt.Sprintf("%d", ts)

	lowerCPU, err := strconv.ParseFloat(lower[ColumnsPS["CpuUsageFloat64"]], 64)
	if err != nil {
		return nil, err
	}
	lowerVmRSS, err := strconv.ParseUint(lower[ColumnsPS["VmRSSBytes"]], 10, 64)
	if err != nil {
		return nil, err
	}
	lowerVmSize, err := strconv.ParseUint(lower[ColumnsPS["VmSizeBytes"]], 10, 64)
	if err != nil {
		return nil, err
	}

	upperCPU, err := strconv.ParseFloat(upper[ColumnsPS["CpuUsageFloat64"]], 64)
	if err != nil {
		return nil, err
	}
	upperVmRSS, err := strconv.ParseUint(upper[ColumnsPS["VmRSSBytes"]], 10, 64)
	if err != nil {
		return nil, err
	}
	upperVmSize, err := strconv.ParseUint(upper[ColumnsPS["VmSizeBytes"]], 10, 64)
	if err != nil {
		return nil, err
	}

	cpuUsageFloat64 := (lowerCPU + upperCPU) / float64(2)
	cpu := fmt.Sprintf("%3.2f %%", cpuUsageFloat64)

	avgVmRSS := (lowerVmRSS + upperVmRSS) / 2
	vmRSS := humanize.Bytes(avgVmRSS)

	avgVmSize := (lowerVmSize + upperVmSize) / 2
	vmSize := humanize.Bytes(avgVmSize)

	nrow[ColumnsPS["CPU"]] = cpu
	nrow[ColumnsPS["CpuUsageFloat64"]] = fmt.Sprintf("%3.2f", cpuUsageFloat64)

	nrow[ColumnsPS["VM_RSS"]] = vmRSS
	nrow[ColumnsPS["VmRSSBytes"]] = fmt.Sprintf("%d", avgVmRSS)

	nrow[ColumnsPS["VM_SIZE"]] = vmSize
	nrow[ColumnsPS["VmSizeBytes"]] = fmt.Sprintf("%d", avgVmSize)

	return nrow, nil
}
