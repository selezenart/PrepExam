package main

import (
	"bytes"
	"debug/macho"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// buildDarwin cross-compiles a tiny Go program for one Mac architecture.
func buildDarwin(t *testing.T, arch string) []byte {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "bin")
	cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", out, src)
	cmd.Env = append(os.Environ(), "GOOS=darwin", "GOARCH="+arch, "CGO_ENABLED=0", "GO111MODULE=off")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building for darwin/%s: %v\n%s", arch, err, b)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestFatHoldsEverySliceIntact(t *testing.T) {
	arm, amd := buildDarwin(t, "arm64"), buildDarwin(t, "amd64")
	fat, err := Fat([][]byte{amd, arm})
	if err != nil {
		t.Fatalf("Fat() error = %v", err)
	}
	f, err := macho.NewFatFile(bytes.NewReader(fat))
	if err != nil {
		t.Fatalf("result is not a fat Mach-O: %v", err)
	}
	want := map[macho.Cpu][]byte{macho.CpuArm64: arm, macho.CpuAmd64: amd}
	if len(f.Arches) != len(want) {
		t.Fatalf("got %d slices, want %d", len(f.Arches), len(want))
	}
	for _, a := range f.Arches {
		orig, ok := want[a.Cpu]
		if !ok {
			t.Fatalf("unexpected slice for %v", a.Cpu)
		}
		if a.Offset%(1<<a.Align) != 0 {
			t.Errorf("%v slice at offset %d is not aligned to 2^%d", a.Cpu, a.Offset, a.Align)
		}
		if !bytes.Equal(fat[a.Offset:a.Offset+a.Size], orig) {
			t.Errorf("%v slice differs from the input binary", a.Cpu)
		}
	}
}

func TestFatRefusesTwoSlicesForOneArchitecture(t *testing.T) {
	arm := buildDarwin(t, "arm64")
	if _, err := Fat([][]byte{arm, arm}); err == nil {
		t.Error("Fat() accepted two arm64 slices; a Mac could not choose between them")
	}
}
