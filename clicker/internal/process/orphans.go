package process

import (
	"path/filepath"
	"strconv"
	"strings"
)

// procEntry is one row of the process table.
type procEntry struct {
	pid  int
	ppid int
	cmd  string
}

// parseProcTable parses `ps -axo pid=,ppid=,command=` output.
func parseProcTable(out string) []procEntry {
	var entries []procEntry
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, err1 := strconv.Atoi(fields[0])
		ppid, err2 := strconv.Atoi(fields[1])
		if err1 != nil || err2 != nil {
			continue
		}
		entries = append(entries, procEntry{pid: pid, ppid: ppid, cmd: strings.Join(fields[2:], " ")})
	}
	return entries
}

// isVibiumCmd reports whether a command line runs the vibium binary. It looks
// at the executable's base name only: matching the whole line would also hit
// unrelated processes whose paths or arguments merely mention vibium (a user
// named vibiumdev, an editor with the repo open).
func isVibiumCmd(cmd string) bool {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return false
	}
	base := filepath.Base(fields[0])
	return base == "vibium" || base == "vibium.exe"
}

// orphanPIDs returns the pids of processes whose command line references
// pathMarker (vibium's browser cache dir) and that have no live vibium
// ancestor. Only vibium launches browsers out of that directory, so a
// chromedriver or Chrome without a vibium ancestor lost its owner (killed
// agent, crashed daemon) and nothing will ever clean it up. Enough of these
// accumulate and browser_start starves (#382).
//
// The ancestry walk is what keeps concurrent sessions safe: another live
// vibium's chromedriver has that vibium as its parent, and its Chrome and
// helper processes reach it transitively.
func orphanPIDs(entries []procEntry, pathMarker string, selfPID int) []int {
	byPid := make(map[int]procEntry, len(entries))
	for _, e := range entries {
		byPid[e.pid] = e
	}

	hasVibiumAncestor := func(e procEntry) bool {
		seen := map[int]bool{}
		for cur := e; !seen[cur.pid]; {
			seen[cur.pid] = true
			parent, ok := byPid[cur.ppid]
			if !ok {
				return false
			}
			if isVibiumCmd(parent.cmd) {
				return true
			}
			cur = parent
		}
		return false
	}

	var out []int
	for _, e := range entries {
		if e.pid <= 1 || e.pid == selfPID {
			continue
		}
		if !strings.Contains(e.cmd, pathMarker) {
			continue
		}
		if hasVibiumAncestor(e) {
			continue
		}
		out = append(out, e.pid)
	}
	return out
}
