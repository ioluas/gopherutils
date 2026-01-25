package size

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
)

func FormatSize(size int64, unit int64) string {
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

func ParseBlockSize(raw string) (config.BlockSizeSpec, string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return config.BlockSizeSpec{}, "missing SIZE", false
	}

	lower := strings.ToLower(trimmed)
	if lower == "human-readable" {
		return config.BlockSizeSpec{Mode: config.BlockSizeModeHuman}, "", true
	}
	if lower == "si" {
		return config.BlockSizeSpec{Mode: config.BlockSizeModeSI}, "", true
	}

	spec := config.BlockSizeSpec{Mode: config.BlockSizeModeBytes}
	if strings.HasPrefix(trimmed, "'") {
		spec.GroupThousands = true
		trimmed = strings.TrimPrefix(trimmed, "'")
	}
	if trimmed == "" {
		return config.BlockSizeSpec{}, "missing SIZE", false
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
		spec.ShowSuffix = true
	} else {
		var err error
		num, err = ParseUintStrict(numStr)
		if err != nil || num == 0 {
			return config.BlockSizeSpec{}, "invalid SIZE number", false
		}
	}

	multiplier, ok := blockSizeMultiplier(suffix)
	if !ok {
		if suffix == "" {
			multiplier = 1
		} else {
			return config.BlockSizeSpec{}, "unknown SIZE suffix", false
		}
	}

	if num > ^uint64(0)/multiplier {
		return config.BlockSizeSpec{}, "SIZE too large", false
	}
	total := num * multiplier
	//goland:noinspection GoRedundantConversion
	const maxInt64 = uint64(^uint64(0) >> 1)
	if total > maxInt64 {
		return config.BlockSizeSpec{}, "SIZE too large", false
	}

	spec.SizeBytes = int64(total)
	spec.Suffix = suffix
	return spec, "", true
}

func ParseUintStrict(s string) (uint64, error) {
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

func FormatSizeWithBlockSpec(size int64, spec config.BlockSizeSpec) string {
	switch spec.Mode {
	case config.BlockSizeModeHuman:
		return FormatSize(size, 1024)
	case config.BlockSizeModeSI:
		return FormatSize(size, 1000)
	default:
	}

	if spec.SizeBytes <= 0 {
		return fmt.Sprintf("%d", size)
	}
	blocks := int64(0)
	if size > 0 {
		blocks = (size-1)/spec.SizeBytes + 1
	}
	out := fmt.Sprintf("%d", blocks)
	if spec.GroupThousands && shouldGroupThousands() {
		out = groupThousands(out)
	}
	if spec.ShowSuffix && spec.Suffix != "" {
		out += spec.Suffix
	}
	return out
}

func shouldGroupThousands() bool {
	for _, envVar := range []string{"LC_ALL", "LC_NUMERIC", "LANG"} {
		if locale := os.Getenv(envVar); locale != "" {
			return locale != "C" && !strings.HasPrefix(locale, "C.") && locale != "POSIX" && !strings.HasPrefix(locale, "POSIX.")
		}
	}
	return false
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
