package paths

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"runtime"
)

// binaryArchMatches reports whether the executable at path targets the running
// architecture. A cached Chrome for Testing directory is trusted by path alone,
// so a wrong-architecture binary (for example the x86 linux64 build a pre-#681
// release's npm postinstall downloads on arm64) passes the stat-based install
// check and then dies at launch with "exec format error" (#699). Treating a
// mismatch as not-installed lets the ensure-install path download the right
// build (resolveVersionDir deletes the rejected directory first).
//
// A file that cannot be parsed as the platform's executable format is treated
// as matching: the stat checks already gate existence, and a parser quirk must
// not discard an otherwise valid install. Only a positively-identified wrong
// architecture is rejected. An architecture vibium is not built for likewise
// returns true, since there is nothing to compare against.
func binaryArchMatches(path string) bool {
	switch runtime.GOOS {
	case "linux":
		f, err := elf.Open(path)
		if err != nil {
			return true
		}
		defer f.Close()
		switch runtime.GOARCH {
		case "amd64":
			return f.Machine == elf.EM_X86_64
		case "arm64":
			return f.Machine == elf.EM_AARCH64
		}
	case "windows":
		f, err := pe.Open(path)
		if err != nil {
			return true
		}
		defer f.Close()
		switch runtime.GOARCH {
		case "amd64":
			return f.Machine == pe.IMAGE_FILE_MACHINE_AMD64
		case "arm64":
			return f.Machine == pe.IMAGE_FILE_MACHINE_ARM64
		}
	case "darwin":
		// Chrome for Testing ships a single-arch Mach-O per platform. A
		// universal (fat) binary fails macho.Open and so is treated as
		// matching, which is the conservative choice.
		f, err := macho.Open(path)
		if err != nil {
			return true
		}
		defer f.Close()
		switch runtime.GOARCH {
		case "amd64":
			return f.Cpu == macho.CpuAmd64
		case "arm64":
			return f.Cpu == macho.CpuArm64
		}
	}
	return true
}
