// Copyright 2016 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package analyze

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/plotutil"
	"github.com/gonum/plot/vg"
	"github.com/gyuho/dataframe"
	"github.com/gyuho/psn/ps"
	"github.com/spf13/cobra"
)

var (
	Command = &cobra.Command{
		Use:   "analyze",
		Short: "Analyzes test results specific to dbtester.",
		RunE:  CommandFunc,
	}
	configPath string
)

func init() {
	Command.PersistentFlags().StringVarP(&configPath, "config", "c", "", "YAML configuration file path.")
}

func CommandFunc(cmd *cobra.Command, args []string) error {
	cfg, err := ReadConfig(configPath)
	if err != nil {
		return err
	}

	println()
	log.Println("Step 1: aggregating each database...")
	for step1Idx, elem := range cfg.Step1 {
		var (
			frames               = []dataframe.Frame{}
			maxCommonMinUnixTime int64
			maxCommonMaxUnixTime int64
		)
		for i, monitorPath := range elem.DataPathList {
			log.Printf("Step 1-%d-%d: creating dataframe from %s", step1Idx, i, monitorPath)

			// fill in missing timestamps
			tb, err := ps.ReadCSVFillIn(monitorPath)
			if err != nil {
				return err
			}
			ext := filepath.Ext(monitorPath)
			cPath := strings.Replace(monitorPath, ext, "-filled-in"+ext, -1)
			if err := tb.ToCSV(cPath); err != nil {
				return err
			}

			fr, err := dataframe.NewFromCSV(nil, cPath)
			if err != nil {
				return err
			}
			nf := dataframe.New()
			c1, err := fr.GetColumn("unix_ts")
			if err != nil {
				return err
			}
			c2, err := fr.GetColumn("CpuUsageFloat64")
			if err != nil {
				return err
			}
			c3, err := fr.GetColumn("VmRSSBytes")
			if err != nil {
				return err
			}
			if err = nf.AddColumn(c1); err != nil {
				return err
			}
			if err = nf.AddColumn(c2); err != nil {
				return err
			}
			if err = nf.AddColumn(c3); err != nil {
				return err
			}
			frames = append(frames, nf)

			fv, ok := c1.FrontNonNil()
			if !ok {
				return fmt.Errorf("FrontNonNil %s has empty Unix time %v", monitorPath, fv)
			}
			fs, ok := fv.ToString()
			if !ok {
				return fmt.Errorf("cannot ToString %v", fv)
			}
			fd, err := strconv.ParseInt(fs, 10, 64)
			if err != nil {
				return err
			}
			bv, ok := c1.BackNonNil()
			if !ok {
				return fmt.Errorf("BackNonNil %s has empty Unix time %v", monitorPath, fv)
			}
			bs, ok := bv.ToString()
			if !ok {
				return fmt.Errorf("cannot ToString %v", bv)
			}
			bd, err := strconv.ParseInt(bs, 10, 64)
			if err != nil {
				return err
			}

			if i == 0 {
				maxCommonMinUnixTime = fd
				maxCommonMaxUnixTime = bd
			}
			if maxCommonMinUnixTime < fd {
				maxCommonMinUnixTime = fd
			}
			if maxCommonMaxUnixTime > bd {
				maxCommonMaxUnixTime = bd
			}
		}

		// make all columns have equal row number, based on the column unix_ts
		// truncate all rows before maxCommonMinUnixTime and after maxCommonMinUnixTime
		minTS := fmt.Sprintf("%d", maxCommonMinUnixTime)
		maxTS := fmt.Sprintf("%d", maxCommonMaxUnixTime)
		frMonitor := dataframe.New()
		for i := range frames {
			uc, err := frames[i].GetColumn("unix_ts")
			if err != nil {
				return err
			}
			j, ok := uc.FindValue(dataframe.NewStringValue(minTS))
			if !ok {
				return fmt.Errorf("%v does not exist in %s", minTS, elem.DataPathList[i])
			}
			k, ok := uc.FindValue(dataframe.NewStringValue(maxTS))
			if !ok {
				return fmt.Errorf("%v does not exist in %s", maxTS, elem.DataPathList[i])
			}

			for _, hd := range frames[i].GetHeader() {
				if i > 0 && hd == "unix_ts" {
					continue
				}
				var col dataframe.Column
				col, err = frames[i].GetColumn(hd)
				if err != nil {
					return err
				}
				if err = col.KeepRows(j, k+1); err != nil {
					return err
				}
				if hd != "unix_ts" {
					switch hd {
					case "CpuUsageFloat64":
						hd = "cpu"
					case "VmRSSBytes":
						hd = "memory_mb"

						// to bytes to mb
						colN := col.RowNumber()
						for rowIdx := 0; rowIdx < colN; rowIdx++ {
							var rowV dataframe.Value
							rowV, err = col.GetValue(rowIdx)
							if err != nil {
								return err
							}
							fv, _ := rowV.ToNumber()
							frv := float64(fv) * 0.000001
							if err = col.SetValue(rowIdx, dataframe.NewStringValue(fmt.Sprintf("%.2f", frv))); err != nil {
								return err
							}
						}
					}
					col.UpdateHeader(fmt.Sprintf("%s_%d", hd, i+1))
				}
				if err = frMonitor.AddColumn(col); err != nil {
					return err
				}
			}
		}

		log.Printf("Step 1-%d-%d: creating dataframe from %s", step1Idx, len(elem.DataPathList), elem.DataBenchmarkPath)
		colMonitorUnixTs, err := frMonitor.GetColumn("unix_ts")
		if err != nil {
			return err
		}
		// need to combine frMonitor to frBench
		frBench, err := dataframe.NewFromCSV(nil, elem.DataBenchmarkPath)
		if err != nil {
			return err
		}
		colBenchUnixTs, err := frBench.GetColumn("unix_ts")
		if err != nil {
			return err
		}

		fv, ok := colBenchUnixTs.FrontNonNil()
		if !ok {
			return fmt.Errorf("FrontNonNil %s has empty Unix time %v", elem.DataBenchmarkPath, fv)
		}
		startRowMonitor, ok := colMonitorUnixTs.FindValue(fv)
		if !ok {
			return fmt.Errorf("%v is not found in monitor results %q", fv, elem.DataPathList)
		}
		bv, ok := colBenchUnixTs.BackNonNil()
		if !ok {
			return fmt.Errorf("BackNonNil %s has empty Unix time %v", elem.DataBenchmarkPath, bv)
		}
		endRowMonitor, ok := colMonitorUnixTs.FindValue(bv)
		if !ok { // monitor short of rows
			endRowMonitor = colMonitorUnixTs.RowNumber() - 1
		}

		var benchLastIdx int
		for _, col := range frBench.GetColumns() {
			if benchLastIdx == 0 {
				benchLastIdx = col.RowNumber()
			}
			if benchLastIdx > col.RowNumber() {
				benchLastIdx = col.RowNumber()
			}
		}
		benchLastIdx--

		if benchLastIdx+1 < endRowMonitor-startRowMonitor+1 { // benchmark is short of rows
			endRowMonitor = startRowMonitor + benchLastIdx
		} else { // monitor is short of rows
			benchLastIdx = endRowMonitor - startRowMonitor
		}

		for _, hd := range frMonitor.GetHeader() {
			if hd == "unix_ts" {
				continue
			}
			var col dataframe.Column
			col, err = frMonitor.GetColumn(hd)
			if err != nil {
				return err
			}
			if err = col.KeepRows(startRowMonitor, endRowMonitor); err != nil {
				return err
			}
			if err = frBench.AddColumn(col); err != nil {
				return err
			}
		}

		log.Printf("Step 1-%d-%d: calculating average values", step1Idx, len(elem.DataPathList)+1)
		var (
			sampleSize              = float64(len(elem.DataPathList))
			cumulativeThroughputCol = dataframe.NewColumn("cumulative_throughput")
			totalThrougput          int
			avgCpuCol               = dataframe.NewColumn("avg_cpu")
			avgMemCol               = dataframe.NewColumn("avg_memory_mb")
		)
		for i := 0; i < benchLastIdx; i++ {
			var (
				cpuTotal    float64
				memoryTotal float64
			)
			for _, col := range frBench.GetColumns() {
				var rv dataframe.Value
				rv, err = col.GetValue(i)
				if err != nil {
					return err
				}
				fv, _ := rv.ToNumber()
				switch {
				case strings.HasPrefix(col.GetHeader(), "cpu_"):
					cpuTotal += fv
				case strings.HasPrefix(col.GetHeader(), "memory_"):
					memoryTotal += fv
				case col.GetHeader() == "throughput":
					fv, _ := rv.ToNumber()
					totalThrougput += int(fv)
					cumulativeThroughputCol.PushBack(dataframe.NewStringValue(totalThrougput))
				}
			}
			avgCpuCol.PushBack(dataframe.NewStringValue(fmt.Sprintf("%.2f", cpuTotal/sampleSize)))
			avgMemCol.PushBack(dataframe.NewStringValue(fmt.Sprintf("%.2f", memoryTotal/sampleSize)))
		}

		log.Printf("Step 1-%d-%d: combine %s and %q", step1Idx, len(elem.DataPathList)+2, elem.DataBenchmarkPath, elem.DataPathList)
		unixTsCol, err := frBench.GetColumn("unix_ts")
		if err != nil {
			return err
		}
		latencyCol, err := frBench.GetColumn("avg_latency_ms")
		if err != nil {
			return err
		}
		throughputCol, err := frBench.GetColumn("throughput")
		if err != nil {
			return err
		}

		aggFr := dataframe.New()
		aggFr.AddColumn(unixTsCol)
		aggFr.AddColumn(latencyCol)
		aggFr.AddColumn(throughputCol)
		aggFr.AddColumn(cumulativeThroughputCol)
		for _, hd := range frBench.GetHeader() {
			col, err := frBench.GetColumn(hd)
			if err != nil {
				return err
			}
			switch {
			case strings.HasPrefix(hd, "cpu_"):
				aggFr.AddColumn(col)
			case strings.HasPrefix(hd, "memory_"):
				aggFr.AddColumn(col)
			}
		}
		aggFr.AddColumn(avgCpuCol)
		aggFr.AddColumn(avgMemCol)

		log.Printf("Step 1-%d-%d: saving to %s", step1Idx, len(elem.DataPathList)+3, elem.OutputPath)
		if err := aggFr.ToCSV(elem.OutputPath); err != nil {
			return err
		}
		println()
	}

	println()
	log.Println("Step 2: aggregating aggregates...")
	for step2Idx, elem := range cfg.Step2 {
		var (
			frames  = []dataframe.Frame{}
			maxSize int
		)
		for i, data := range elem.DataList {
			log.Printf("Step 2-%d-%d: creating dataframe from %s...", step2Idx, i, data.Path)
			fr, err := dataframe.NewFromCSV(nil, data.Path)
			if err != nil {
				return err
			}
			frames = append(frames, fr)

			col, err := fr.GetColumn("unix_ts")
			if err != nil {
				return err
			}
			rNum := col.RowNumber()
			if maxSize < rNum {
				maxSize = rNum
			}
		}

		nf := dataframe.New()
		secondCol := dataframe.NewColumn("second")
		for i := 0; i < maxSize; i++ {
			secondCol.PushBack(dataframe.NewStringValue(i))
		}
		nf.AddColumn(secondCol)
		colsToKeep := []string{"avg_latency_ms", "throughput", "cumulative_throughput", "avg_cpu", "avg_memory_mb"}
		for i, fr := range frames {
			dbID := elem.DataList[i].Name
			log.Printf("Step 2-%d-%d: cleaning up %s...", step2Idx, i, dbID)
			for _, col := range fr.GetColumns() {
				toSkip := true
				for _, cv := range colsToKeep {
					if col.GetHeader() == cv {
						toSkip = false
						break
					}
				}
				if toSkip {
					continue
				}
				if err := col.Appends(dataframe.NewStringValueNil(), maxSize); err != nil {
					return err
				}
				col.UpdateHeader(fmt.Sprintf("%s_%s", col.GetHeader(), dbID))
				nf.AddColumn(col)
			}
		}

		log.Printf("Step 2-%d: saving to %s...", step2Idx, elem.OutputPath)
		if err := nf.ToCSV(elem.OutputPath); err != nil {
			return err
		}
	}

	println()
	log.Println("Step 3: plotting...")

	plot.DefaultFont = "Helvetica"
	plotter.DefaultLineStyle.Width = vg.Points(1.5)
	plotter.DefaultGlyphStyle.Radius = vg.Points(2.0)
	var (
		plotWidth  = 12 * vg.Inch
		plotHeight = 8 * vg.Inch
	)
	for step3Idx, elem := range cfg.Step3 {
		fr, err := dataframe.NewFromCSV(nil, elem.DataPath)
		if err != nil {
			return err
		}
		log.Printf("Step 3-%d: %s with %q", step3Idx, elem.DataPath, fr.GetHeader())

		for i, pelem := range elem.PlotList {
			log.Printf("Step 3-%d-%d: %s at %q", step3Idx, i, pelem.YAxis, pelem.OutputPathList)
			pl, err := plot.New()
			if err != nil {
				return err
			}
			pl.Title.Text = fmt.Sprintf("%s, %s", cfg.Titles[step3Idx], pelem.YAxis)
			pl.X.Label.Text = pelem.XAxis
			pl.Y.Label.Text = pelem.YAxis
			pl.Legend.Top = true

			var args []interface{}
			for _, line := range pelem.Lines {
				col, err := fr.GetColumn(line.Column)
				if err != nil {
					return err
				}
				pt, err := points(col)
				if err != nil {
					return err
				}
				args = append(args, line.Legend, pt)
			}

			if err = plotutil.AddLines(pl, args...); err != nil {
				return err
			}
			for _, outputPath := range pelem.OutputPathList {
				if err = pl.Save(plotWidth, plotHeight, outputPath); err != nil {
					return err
				}
			}
		}
	}

	println()
	log.Println("Step 4: writing README...")
	rdBuf := new(bytes.Buffer)
	rdBuf.WriteString("\n\n")
	rdBuf.WriteString(cfg.Step4.Preface)
	rdBuf.WriteString("\n\n\n")
	for i, result := range cfg.Step4.Results {
		rdBuf.WriteString(fmt.Sprintf("<br><br><hr>\n##### %s", cfg.Titles[i]))
		rdBuf.WriteString("\n\n")
		for _, img := range result.Images {
			imgPath := ""
			switch img.ImageType {
			case "local":
				imgPath = "./" + filepath.Base(img.ImagePath)
				rdBuf.WriteString(fmt.Sprintf("![%s](%s)\n\n", img.ImageTitle, imgPath))
			case "remote":
				rdBuf.WriteString(fmt.Sprintf(`<img src="%s" alt="%s">`, img.ImagePath, img.ImageTitle))
				rdBuf.WriteString("\n\n")
			default:
				return fmt.Errorf("%s is not supported", img.ImageType)
			}
		}
		rdBuf.WriteString("\n\n")
	}
	if err := toFile(rdBuf.String(), cfg.Step4.OutputPath); err != nil {
		return err
	}

	println()
	log.Println("FINISHED!")
	return nil
}

func toFile(txt, fpath string) error {
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		f, err = os.Create(fpath)
		if err != nil {
			return err
		}
	}
	defer f.Close()
	if _, err := f.WriteString(txt); err != nil {
		return err
	}
	return nil
}

func points(col dataframe.Column) (plotter.XYs, error) {
	bv, ok := col.BackNonNil()
	if !ok {
		return nil, fmt.Errorf("BackNonNil not found")
	}
	rowN, ok := col.FindValue(bv)
	if !ok {
		return nil, fmt.Errorf("not found %v", bv)
	}
	pts := make(plotter.XYs, rowN)
	for i := range pts {
		v, err := col.GetValue(i)
		if err != nil {
			return nil, err
		}
		n, _ := v.ToNumber()
		pts[i].X = float64(i)
		pts[i].Y = n
	}
	return pts, nil
}
