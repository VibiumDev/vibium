package process

import (
	"reflect"
	"testing"
)

const marker = "/Users/u/Library/Caches/vibium/chrome-for-testing"

func TestParseProcTable(t *testing.T) {
	out := "  101     1 /usr/sbin/thing --flag\n" +
		"  202   101 /Users/u/Library/Caches/vibium/chrome-for-testing/152.0/chromedriver --port=9515\n" +
		"garbage line\n" +
		"  303   202 /path with spaces/Google Chrome for Testing --type=renderer\n"
	got := parseProcTable(out)
	want := []procEntry{
		{pid: 101, ppid: 1, cmd: "/usr/sbin/thing --flag"},
		{pid: 202, ppid: 101, cmd: "/Users/u/Library/Caches/vibium/chrome-for-testing/152.0/chromedriver --port=9515"},
		{pid: 303, ppid: 202, cmd: "/path with spaces/Google Chrome for Testing --type=renderer"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseProcTable:\ngot  %+v\nwant %+v", got, want)
	}
}

func TestOrphanPIDs(t *testing.T) {
	cd := marker + "/152.0/chromedriver --port=9515"
	chrome := marker + "/152.0/chrome-mac-arm64/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing --remote-debugging-port=0"

	tests := []struct {
		name    string
		entries []procEntry
		want    []int
	}{
		{
			name: "chromedriver reparented to init is an orphan, and so is its chrome",
			entries: []procEntry{
				{pid: 1, ppid: 0, cmd: "/sbin/launchd"},
				{pid: 10, ppid: 1, cmd: cd},
				{pid: 11, ppid: 10, cmd: chrome},
			},
			want: []int{10, 11},
		},
		{
			name: "chain owned by a live vibium survives",
			entries: []procEntry{
				{pid: 1, ppid: 0, cmd: "/sbin/launchd"},
				{pid: 5, ppid: 1, cmd: "/usr/local/bin/vibium serve"},
				{pid: 10, ppid: 5, cmd: cd},
				{pid: 11, ppid: 10, cmd: chrome},
			},
			want: nil,
		},
		{
			name: "another tool's chromedriver outside the cache dir is ignored",
			entries: []procEntry{
				{pid: 1, ppid: 0, cmd: "/sbin/launchd"},
				{pid: 20, ppid: 1, cmd: "/opt/selenium/chromedriver --port=4444"},
			},
			want: nil,
		},
		{
			name: "vibium in an unrelated path does not confer ownership",
			entries: []procEntry{
				{pid: 1, ppid: 0, cmd: "/sbin/launchd"},
				// e.g. a shell or editor belonging to a user named vibiumdev
				{pid: 30, ppid: 1, cmd: "/bin/zsh /Users/vibiumdev/run.sh"},
				{pid: 31, ppid: 30, cmd: cd},
			},
			want: []int{31},
		},
		{
			name: "orphan reparented to a subreaper that is not vibium",
			entries: []procEntry{
				{pid: 1, ppid: 0, cmd: "/sbin/launchd"},
				{pid: 40, ppid: 1, cmd: "/usr/lib/systemd/systemd --user"},
				{pid: 41, ppid: 40, cmd: cd},
			},
			want: []int{41},
		},
		{
			name: "ppid cycle terminates without owning anyone",
			entries: []procEntry{
				{pid: 50, ppid: 51, cmd: cd},
				{pid: 51, ppid: 50, cmd: "/bin/thing"},
			},
			want: []int{50},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := orphanPIDs(tt.entries, marker, 999999)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrphanPIDsSkipsSelf(t *testing.T) {
	entries := []procEntry{
		{pid: 60, ppid: 1, cmd: marker + "/152.0/chromedriver"},
	}
	if got := orphanPIDs(entries, marker, 60); got != nil {
		t.Fatalf("self pid must never be reaped, got %v", got)
	}
}

func TestIsVibiumCmd(t *testing.T) {
	yes := []string{"vibium serve", "/usr/local/bin/vibium daemon"}
	no := []string{"", "/bin/zsh", "/Users/vibiumdev/notes vibium", "vibium-helper --x"}
	for _, c := range yes {
		if !isVibiumCmd(c) {
			t.Errorf("isVibiumCmd(%q) = false, want true", c)
		}
	}
	for _, c := range no {
		if isVibiumCmd(c) {
			t.Errorf("isVibiumCmd(%q) = true, want false", c)
		}
	}
}
