package paths

import (
	"io"
	"os"
	"runtime"
)

// elfMachines maps GOARCH to the ELF e_machine value a binary for that
// architecture carries.
var elfMachines = map[string]uint16{
	"amd64": 0x3e, // EM_X86_64
	"arm64": 0xb7, // EM_AARCH64
}

// binaryArchMatches reports whether the binary at path can run on this
// machine. Only Linux ELF binaries are checked; anything unreadable or
// non-ELF is trusted. The check exists to reject the one observed failure,
// an x86-64 Chrome download cached on arm64 Linux, which IsInstalled
// certified and the launch then hit as an exec format error (#699). It is
// not a re-verification of healthy installs, so it never blocks one.
func binaryArchMatches(path string) bool {
	if runtime.GOOS != "linux" {
		return true
	}
	return elfMachineMatches(path, runtime.GOARCH)
}

// elfMachineMatches reports whether the file at path is an ELF binary built
// for goarch. Unknown architectures, unreadable files, and non-ELF content
// return true (trusted), so only a positively identified mismatch rejects.
func elfMachineMatches(path, goarch string) bool {
	want, ok := elfMachines[goarch]
	if !ok {
		return true
	}
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()

	var h [20]byte
	if _, err := io.ReadFull(f, h[:]); err != nil {
		return true
	}
	if h[0] != 0x7f || h[1] != 'E' || h[2] != 'L' || h[3] != 'F' {
		return true
	}
	// e_machine sits at offset 18 in the byte order EI_DATA declares.
	machine := uint16(h[18]) | uint16(h[19])<<8
	if h[5] == 2 { // ELFDATA2MSB
		machine = uint16(h[19]) | uint16(h[18])<<8
	}
	return machine == want
}
