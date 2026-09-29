// Command lipo joins single-architecture Mach-O executables into one
// universal ("fat") binary, like Apple's lipo -create. It exists so the
// release can ship one macOS file that runs natively on both Intel and
// Apple Silicon, built from Linux where Apple's lipo is not available.
//
// Usage: go run ./tools/lipo -o out in1 in2 ...
package main

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
)

// sliceAlign is the log2 alignment of every slice: 16 KiB, the page size on
// Apple Silicon and what Apple's lipo uses for arm64 and x86_64 alike.
const sliceAlign = 14

// fatMagic identifies a fat Mach-O header, stored big-endian.
const fatMagic = 0xcafebabe

// Fat combines Mach-O executables, one per CPU type, into a fat binary. Each
// input is copied unchanged, so any code signature it carries stays valid.
func Fat(slices [][]byte) ([]byte, error) {
	type arch struct {
		cpu, sub uint32
		data     []byte
	}
	var arches []arch
	seen := map[macho.Cpu]bool{}
	for i, b := range slices {
		f, err := macho.NewFile(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("input %d is not a thin Mach-O file: %w", i+1, err)
		}
		if seen[f.Cpu] {
			return nil, fmt.Errorf("two inputs are both %v; each architecture may appear once", f.Cpu)
		}
		seen[f.Cpu] = true
		arches = append(arches, arch{uint32(f.Cpu), f.SubCpu, b})
	}

	const headerSize, archSize = 8, 20
	align := uint32(1) << sliceAlign
	offset := roundUp(headerSize+archSize*uint32(len(arches)), align)

	var out bytes.Buffer
	be := binary.BigEndian
	put := func(v uint32) { _ = binary.Write(&out, be, v) }
	put(fatMagic)
	put(uint32(len(arches)))
	offsets := make([]uint32, len(arches))
	for i, a := range arches {
		offsets[i] = offset
		put(a.cpu)
		put(a.sub)
		put(offset)
		put(uint32(len(a.data)))
		put(sliceAlign)
		offset = roundUp(offset+uint32(len(a.data)), align)
	}
	for i, a := range arches {
		out.Write(make([]byte, int(offsets[i])-out.Len()))
		out.Write(a.data)
	}
	return out.Bytes(), nil
}

func roundUp(n, align uint32) uint32 { return (n + align - 1) / align * align }

func main() {
	output := flag.String("o", "", "path of the universal binary to write")
	flag.Parse()
	if *output == "" || flag.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "usage: lipo -o out in1 in2 ...")
		os.Exit(2)
	}
	var slices [][]byte
	for _, path := range flag.Args() {
		b, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "lipo:", err)
			os.Exit(1)
		}
		slices = append(slices, b)
	}
	fat, err := Fat(slices)
	if err != nil {
		fmt.Fprintln(os.Stderr, "lipo:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, fat, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "lipo:", err)
		os.Exit(1)
	}
}
