package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func formatSize(size int64, unit int64) string {
	if size < unit {
		return fmt.Sprintf("%d", size)
	}
	const suffixes = "KMGTPE"
	const maxExp = len(suffixes) - 1
	div, exp := unit, 0
	for n := size / unit; n >= unit && exp < maxExp; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(size)/float64(div), suffixes[exp])
}

func parseBlockSize(raw string) (BlockSizeSpec, string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return BlockSizeSpec{}, "missing SIZE", false
	}

	lower := strings.ToLower(trimmed)
	if lower == "human-readable" {
		return BlockSizeSpec{mode: blockSizeModeHuman}, "", true
	}
	if lower == "si" {
		return BlockSizeSpec{mode: blockSizeModeSI}, "", true
	}

	spec := BlockSizeSpec{mode: blockSizeModeBytes}
	if strings.HasPrefix(trimmed, "'") {
		spec.groupThousands = true
		trimmed = strings.TrimPrefix(trimmed, "'")
	}
	if trimmed == "" {
		return BlockSizeSpec{}, "missing SIZE", false
	}

	var numStr string
	var suffix string
	nonDigitIdx := -1
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] < '0' || trimmed[i] > '9' {
			nonDigitIdx = i
			break
		}
	}
	switch nonDigitIdx {
	case -1:
		numStr = trimmed
	case 0:
		suffix = trimmed
	default:
		numStr = trimmed[:nonDigitIdx]
		suffix = trimmed[nonDigitIdx:]
	}

	var num uint64
	if numStr == "" {
		num = 1
		spec.showSuffix = true
	} else {
		var err error
		num, err = parseUintStrict(numStr)
		if err != nil || num == 0 {
			return BlockSizeSpec{}, "invalid SIZE number", false
		}
	}

	multiplier, ok := blockSizeMultiplier(suffix)
	if !ok {
		if suffix == "" {
			multiplier = 1
		} else {
			return BlockSizeSpec{}, "unknown SIZE suffix", false
		}
	}

	if num > ^uint64(0)/multiplier {
		return BlockSizeSpec{}, "SIZE too large", false
	}
	total := num * multiplier
	//goland:noinspection GoRedundantConversion
	const maxInt64 = uint64(^uint64(0) >> 1)
	if total > maxInt64 {
		return BlockSizeSpec{}, "SIZE too large", false
	}

	spec.sizeBytes = int64(total)
	spec.suffix = suffix
	return spec, "", true
}

func parseUintStrict(s string) (uint64, error) {
	if s == "" {
		return 0, errors.New("empty")
	}
	var n uint64
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch < '0' || ch > '9' {
			return 0, errors.New("invalid")
		}
		d := uint64(ch - '0')
		if n > (^uint64(0)-d)/10 {
			return 0, errors.New("overflow")
		}
		n = n*10 + d
	}
	return n, nil
}

var blockSizeBinaryMultipliers = map[string]uint64{
	"k":   1 << 10,
	"K":   1 << 10,
	"KiB": 1 << 10,
	"M":   1 << 20,
	"MiB": 1 << 20,
	"G":   1 << 30,
	"GiB": 1 << 30,
	"T":   1 << 40,
	"TiB": 1 << 40,
	"P":   1 << 50,
	"PiB": 1 << 50,
	"E":   1 << 60,
	"EiB": 1 << 60,
}

var blockSizeDecimalMultipliers = map[string]uint64{
	"kB": 1_000,
	"MB": 1_000_000,
	"GB": 1_000_000_000,
	"TB": 1_000_000_000_000,
	"PB": 1_000_000_000_000_000,
	"EB": 1_000_000_000_000_000_000,
}

func blockSizeMultiplier(suffix string) (uint64, bool) {
	if suffix == "" {
		return 1, true
	}

	if v, ok := blockSizeBinaryMultipliers[suffix]; ok {
		return v, true
	}

	if v, ok := blockSizeDecimalMultipliers[suffix]; ok {
		return v, true
	}

	return 0, false
}

func formatSizeWithBlockSpec(size int64, spec BlockSizeSpec) string {
	switch spec.mode {
	case blockSizeModeHuman:
		return formatSize(size, 1024)
	case blockSizeModeSI:
		return formatSize(size, 1000)
	default:
	}

	if spec.sizeBytes <= 0 {
		return fmt.Sprintf("%d", size)
	}
	blocks := int64(0)
	if size > 0 {
		blocks = (size-1)/spec.sizeBytes + 1
	}
	out := fmt.Sprintf("%d", blocks)
	if spec.groupThousands && shouldGroupThousands() {
		out = groupThousands(out)
	}
	if spec.showSuffix && spec.suffix != "" {
		out += spec.suffix
	}
	return out
}

func shouldGroupThousands() bool {
	locale := os.Getenv("LC_NUMERIC")
	if locale == "" {
		return false
	}
	return locale != "C" && locale != "POSIX"
}

func groupThousands(s string) string {
	if len(s) <= 3 {
		return s
	}
	rem := len(s) % 3
	if rem == 0 {
		rem = 3
	}
	var b strings.Builder
	b.Grow(len(s) + (len(s)-1)/3)
	b.WriteString(s[:rem])
	for i := rem; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
