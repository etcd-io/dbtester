package ps

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gyuho/dataframe"
	"github.com/olekukonko/tablewriter"
)

// ProcessTableColumns is columns for CSV file.
var ProcessTableColumns = []string{
	"NAME",
	"STATE",

	"PID",
	"PPID",

	"CPU",
	"VM_RSS",
	"VM_SIZE",

	"FD",
	"THREADS",

	"CpuUsageFloat64",
	"VmRSSBytes",
	"VmSizeBytes",
}

// List finds the status by specifying the filter.
func List(filter *Process) ([]Process, error) {
	if filter != nil {
		if filter.Stat.Pid != 0 && filter.Status.Pid == 0 {
			filter.Status.Pid = filter.Stat.Pid
		}
		if filter.Stat.Pid == 0 && filter.Status.Pid != 0 {
			filter.Stat.Pid = filter.Status.Pid
		}
	}
	if filter != nil && filter.Stat.Pid != 0 && filter.Status.Pid != 0 { // no need to scan all 'proc'
		stat, err := GetStat(filter.Stat.Pid)
		if err != nil {
			return nil, err
		}
		status, err := GetStatus(filter.Status.Pid)
		if err != nil {
			return nil, err
		}
		return []Process{Process{Stat: stat, Status: status}}, err
	}

	// scan all 'proc's
	ds, err := ioutil.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	pids := []int64{}
	for _, f := range ds {
		if f.IsDir() && isInt(f.Name()) {
			i, err := strconv.ParseInt(f.Name(), 10, 64)
			if err != nil {
				return nil, err
			}
			pids = append(pids, i)
		}
	}

	donec, errc := make(chan struct{}), make(chan error)
	rmap := make(map[int64]Process)
	var mu sync.Mutex

	for _, pid := range pids {
		go func(pid int64, filter *Process) {
			stat, err := GetStat(pid)
			if err != nil {
				errc <- err
				return
			}
			status, err := GetStatus(pid)
			if err != nil {
				errc <- err
				return
			}
			if stat.Match(filter) && status.Match(filter) {
				mu.Lock()
				rmap[pid] = Process{Stat: stat, Status: status}
				mu.Unlock()
			}
			donec <- struct{}{}
		}(pid, filter)
	}

	cnt := 0
	for cnt != len(pids) {
		select {
		case <-donec:
			cnt++
		case e := <-errc:
			return nil, e
		}
	}

	rs := []Process{}
	for _, proc := range rmap {
		if proc.Stat.Comm == "" && proc.Status.Name != "" {
			proc.Stat.Comm = proc.Status.Name
		}
		if proc.Stat.Comm != "" && proc.Status.Name == "" {
			proc.Status.Name = proc.Stat.Comm
		}
		if proc.Stat.Pid == 0 && proc.Status.Pid != 0 {
			proc.Stat.Pid = proc.Status.Pid
		}
		if proc.Stat.Pid != 0 && proc.Status.Pid == 0 {
			proc.Status.Pid = proc.Stat.Pid
		}
		rs = append(rs, proc)
	}
	return rs, nil
}

// Match matches Stat to filter. Only supports name and PID.
func (s *Stat) Match(filter *Process) bool {
	if s == nil {
		return false
	}
	if filter == nil {
		return true // no need to compare
	}
	name := filter.Stat.Comm
	if name != "" {
		if name != s.Comm {
			return false
		}
	}
	pid := filter.Stat.Pid
	if pid != 0 {
		if pid != s.Pid {
			return false
		}
	}
	state := filter.Stat.State
	if state != "" {
		if state != s.State {
			return false
		}
	}
	return true
}

// Match matches Status to filter. Only supports name and PID.
func (s *Status) Match(filter *Process) bool {
	if s == nil {
		return false
	}
	if filter == nil {
		return true // no need to compare
	}
	name := filter.Status.Name
	if name != "" {
		if name != s.Name {
			return false
		}
	}
	pid := filter.Status.Pid
	if pid != 0 {
		if pid != s.Pid {
			return false
		}
	}
	state := filter.Status.State
	if state != "" {
		if state != s.State {
			return false
		}
	}
	return true
}

// WriteToTable writes slice of Process to ASCII table.
func WriteToTable(w io.Writer, top int, pss ...Process) {
	tw := tablewriter.NewWriter(w)
	tw.SetHeader(ProcessTableColumns[:9:9])

	rows := make([][]string, len(pss))
	for i, s := range pss {
		sl := make([]string, len(ProcessTableColumns))
		sl[0] = s.Status.Name
		sl[1] = s.Status.State
		sl[2] = fmt.Sprintf("%d", s.Status.Pid)
		sl[3] = fmt.Sprintf("%d", s.Status.PPid)
		sl[4] = fmt.Sprintf("%3.2f %%", s.Stat.CpuUsage)
		sl[5] = s.Status.VmRSS
		sl[6] = s.Status.VmSize
		sl[7] = fmt.Sprintf("%d", s.Status.FDSize)
		sl[8] = fmt.Sprintf("%d", s.Status.Threads)

		sl[9] = fmt.Sprintf("%3.2f", s.Stat.CpuUsage)
		sl[10] = fmt.Sprintf("%d", s.Status.VmRSSBytes)
		sl[11] = fmt.Sprintf("%d", s.Status.VmSizeBytes)
		rows[i] = sl
	}
	dataframe.SortBy(
		rows,
		dataframe.NumberDescendingFunc(10), // VM_RSS
		dataframe.NumberDescendingFunc(9),  // CPU
		dataframe.NumberDescendingFunc(11), // VM_SIZE
	).Sort(rows)

	if top != 0 && len(rows) > top {
		rows = rows[:top:top]
	}
	for _, row := range rows {
		tw.Append(row[:9:9])
	}

	tw.Render()
}

var once sync.Once

// WriteToTable writes slice of Process to a csv file.
func WriteToCSV(f *os.File, pss ...Process) error {
	wr := csv.NewWriter(f)

	var werr error
	writeCSVHeader := func() {
		if err := wr.Write(append([]string{"unix_ts"}, ProcessTableColumns...)); err != nil {
			werr = err
		}
	}
	once.Do(writeCSVHeader)
	if werr != nil {
		return werr
	}

	rows := make([][]string, len(pss))
	for i, s := range pss {
		sl := make([]string, len(ProcessTableColumns))
		sl[0] = s.Status.Name
		sl[1] = s.Status.State
		sl[2] = fmt.Sprintf("%d", s.Status.Pid)
		sl[3] = fmt.Sprintf("%d", s.Status.PPid)
		sl[4] = fmt.Sprintf("%3.2f %%", s.Stat.CpuUsage)
		sl[5] = s.Status.VmRSS
		sl[6] = s.Status.VmSize
		sl[7] = fmt.Sprintf("%d", s.Status.FDSize)
		sl[8] = fmt.Sprintf("%d", s.Status.Threads)

		sl[9] = fmt.Sprintf("%3.2f", s.Stat.CpuUsage)
		sl[10] = fmt.Sprintf("%d", s.Status.VmRSSBytes)
		sl[11] = fmt.Sprintf("%d", s.Status.VmSizeBytes)
		rows[i] = sl
	}
	dataframe.SortBy(
		rows,
		dataframe.NumberDescendingFunc(10), // VM_RSS
		dataframe.NumberDescendingFunc(9),  // CPU
		dataframe.NumberDescendingFunc(11), // VM_SIZE
	).Sort(rows)

	ts := fmt.Sprintf("%d", time.Now().Unix())
	nrows := make([][]string, len(rows))
	for i, row := range rows {
		nrows[i] = append([]string{ts}, row...)
	}
	if err := wr.WriteAll(nrows); err != nil {
		return err
	}

	wr.Flush()
	return wr.Error()
}

type int64Slice []int64

func (s int64Slice) Len() int           { return len(s) }
func (s int64Slice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s int64Slice) Less(i, j int) bool { return s[i] < s[j] }

// Kill kills all processes in arguments.
func Kill(w io.Writer, parent bool, pss ...Process) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintln(w, "Kill:", err)
		}
	}()

	pidToKill := make(map[int64]string)
	for _, s := range pss {
		pidToKill[s.Status.Pid] = s.Status.Name
		if parent && s.Status.PPid != 0 {
			pidToKill[s.Status.PPid] = s.Status.Name
		}
	}
	if len(pidToKill) == 0 {
		fmt.Fprintln(w, "no PID to kill...")
		return
	}

	pids := []int64{}
	for pid := range pidToKill {
		pids = append(pids, pid)
	}
	sort.Sort(int64Slice(pids))

	for _, pid := range pids {
		fmt.Fprintf(w, "\nsyscall.Kill: %s [PID: %d]\n", pidToKill[pid], pid)
		if err := syscall.Kill(int(pid), syscall.SIGTERM); err != nil {
			fmt.Fprintf(w, "syscall.SIGTERM error (%v)\n", err)

			shell := os.Getenv("SHELL")
			if len(shell) == 0 {
				shell = "sh"
			}
			args := []string{shell, "-c", fmt.Sprintf("sudo kill -9 %d", pid)}
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Stdout = w
			cmd.Stderr = w
			fmt.Fprintf(w, "Starting: %q\n", strings.Join(cmd.Args, ""))
			if err := cmd.Start(); err != nil {
				fmt.Fprintf(w, "error when 'sudo kill -9' (%v)\n", err)
			}
			if err := cmd.Wait(); err != nil {
				fmt.Fprintf(w, "Start(%s) cmd.Wait returned %v\n", cmd.Path, err)
			}
			fmt.Fprintf(w, "    Done: %q\n", strings.Join(cmd.Args, ""))
		}
		if err := syscall.Kill(int(pid), syscall.SIGKILL); err != nil {
			fmt.Fprintf(w, "syscall.SIGKILL error (%v)\n", err)

			shell := os.Getenv("SHELL")
			if len(shell) == 0 {
				shell = "sh"
			}
			args := []string{shell, "-c", fmt.Sprintf("sudo kill -9 %d", pid)}
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Stdout = w
			cmd.Stderr = w
			fmt.Fprintf(w, "Starting: %q\n", strings.Join(cmd.Args, ""))
			if err := cmd.Start(); err != nil {
				fmt.Fprintf(w, "error when 'sudo kill -9' (%v)\n", err)
			}
			if err := cmd.Wait(); err != nil {
				fmt.Fprintf(w, "Start(%s) cmd.Wait returned %v\n", cmd.Path, err)
			}
			fmt.Fprintf(w, "    Done: %q\n", strings.Join(cmd.Args, ""))
		}
	}
	fmt.Fprintln(w)
}
