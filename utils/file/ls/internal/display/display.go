package display

import (
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/ioluas/gopherutils/internal/fsutil"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/entry"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/size"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/timeutil"
	"golang.org/x/term"
)

func QuoteName(name string) string {
	var b strings.Builder
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch c {
		case '\a':
			b.WriteString(`\a`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\v':
			b.WriteString(`\v`)
		case '\\':
			b.WriteString(`\\`)
		default:
			if c < 32 || c >= 127 {
				_, _ = fmt.Fprintf(&b, "\\%03o", c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	return b.String()
}

func PrintLongList(stdout, stderr io.Writer, entries []os.DirEntry, config *config.Config) bool {
	if len(entries) == 0 {
		return false
	}

	var hadError bool

	// Gather all file info
	details := make([]fileDetails, 0, len(entries))
	var maxLinkLen, maxOwnerLen, maxAuthorLen, maxGroupLen, maxSizeLen int
	for _, dirEntry := range entries {
		if ce, ok := dirEntry.(*entry.CachedDirEntry); ok && ce.Err != nil {
			_, _ = fmt.Fprintf(stderr, "ls: cannot access '%s': %v\n", ce.Name(), ce.Err)
			hadError = true
			continue
		}

		info, err := dirEntry.Info()
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "ls: cannot access '%s': %v\n", dirEntry.Name(), err)
			hadError = true
			continue
		}

		var nlink uint64 = 1
		var owner, group, author string
		fileSize := info.Size()

		sysData := info.Sys()
		sysStat, ok := sysData.(*syscall.Stat_t)
		if ok && sysStat != nil {
			//goland:noinspection GoRedundantConversion
			nlink = uint64(sysStat.Nlink)
			owner, group = fsutil.GetOwnerGroup(sysStat)
			if config.ShowAuthor {
				author = owner
			}
		} else {
			// Non-Unix fallback
			owner = "-" // Placeholder for owner
			group = "-" // Placeholder for group
			if config.ShowAuthor {
				author = "-"
			}
		}

		var sizeStr string
		if config.BlockSize != nil {
			sizeStr = size.FormatSizeWithBlockSpec(fileSize, *config.BlockSize)
		} else if config.SI {
			sizeStr = size.FormatSize(fileSize, 1000)
		} else if config.HumanReadable {
			sizeStr = size.FormatSize(fileSize, 1024)
		} else {
			sizeStr = fmt.Sprintf("%d", fileSize)
		}

		name := dirEntry.Name()
		if config.Escape {
			name = QuoteName(name)
		}

		var entryTime time.Time
		if ce, ok := dirEntry.(*entry.CachedDirEntry); ok && ce.Time != nil {
			entryTime = *ce.Time
		} else {
			entryTime = timeutil.GetEntryTime(info, config.TimeField)
		}

		d := fileDetails{
			mode:    info.Mode().String(),
			nlink:   nlink,
			owner:   owner,
			author:  author,
			group:   group,
			sizeStr: sizeStr,
			modTime: entryTime,
			name:    name,
		}
		details = append(details, d)

		// Calculate max widths
		if l := len(fmt.Sprint(d.nlink)); l > maxLinkLen {
			maxLinkLen = l
		}
		if l := len(d.owner); l > maxOwnerLen {
			maxOwnerLen = l
		}
		if config.ShowAuthor {
			if l := len(d.author); l > maxAuthorLen {
				maxAuthorLen = l
			}
		}
		if !config.NoGroup {
			if l := len(d.group); l > maxGroupLen {
				maxGroupLen = l
			}
		}
		if l := len(d.sizeStr); l > maxSizeLen {
			maxSizeLen = l
		}
	}
	// Ensure minimum width for nlink, as ls often does
	if maxLinkLen < 2 {
		maxLinkLen = 2
	}

	// Print
	// Format: mode nlink owner group size date time name
	// e.g. -rw-r--r-- 1 user group 123 Jan 01 12:00 file.txt
	widths := longListWidths{
		link:   maxLinkLen,
		owner:  maxOwnerLen,
		author: maxAuthorLen,
		group:  maxGroupLen,
		size:   maxSizeLen,
	}
	for _, d := range details {
		format, args := longListFormatArgs(d, widths, config)
		_, _ = fmt.Fprintf(stdout, format, args...)
	}
	return hadError
}

type fileDetails struct {
	mode    string
	nlink   uint64
	owner   string
	author  string
	group   string
	sizeStr string
	modTime time.Time
	name    string
}

type longListWidths struct {
	link   int
	owner  int
	author int
	group  int
	size   int
}

func longListFormatArgs(d fileDetails, widths longListWidths, config *config.Config) (string, []interface{}) {
	timeStr := timeutil.FormatTime(d.modTime, config)
	switch {
	case config.ShowAuthor && config.NoGroup:
		return "%s %*d %-*s %-*s %*s %s %s\n", []interface{}{
			d.mode,
			widths.link, d.nlink,
			widths.owner, d.owner,
			widths.author, d.author,
			widths.size, d.sizeStr,
			timeStr,
			d.name,
		}
	case config.ShowAuthor:
		return "%s %*d %-*s %-*s %-*s %*s %s %s\n", []interface{}{
			d.mode,
			widths.link, d.nlink,
			widths.owner, d.owner,
			widths.author, d.author,
			widths.group, d.group,
			widths.size, d.sizeStr,
			timeStr,
			d.name,
		}
	case config.NoGroup:
		return "%s %*d %-*s %*s %s %s\n", []interface{}{
			d.mode,
			widths.link, d.nlink,
			widths.owner, d.owner,
			widths.size, d.sizeStr,
			timeStr,
			d.name,
		}
	default:
		// Print without author column.
		return "%s %*d %-*s %-*s %*s %s %s\n", []interface{}{
			d.mode,
			widths.link, d.nlink,
			widths.owner, d.owner,
			widths.group, d.group,
			widths.size, d.sizeStr,
			timeStr,
			d.name,
		}
	}
}

// printEntries prints entry names to the output writer.
// It detects if the writer is a terminal to format output in columns.
func PrintEntries(w io.Writer, names []string) {
	if len(names) == 0 {
		return
	}
	// Check if output is a terminal
	var termWidth int
	isTerminal := false
	if f, ok := w.(interface{ Fd() uintptr }); ok {
		fd := int(f.Fd())
		if IsTerminalFunc(fd) {
			width, _, err := GetTermSizeFunc(fd)
			if err == nil {
				termWidth = width
				isTerminal = true
			}
		}
	}

	if isTerminal {
		PrintGrid(w, names, termWidth)
	} else {
		// Not a terminal or failed to get size: one entry per line
		for _, name := range names {
			_, _ = fmt.Fprintln(w, name)
		}
	}
}

// printGrid formats names into a grid that fits within width.
// Users column-major order (standard ls behavior).
func PrintGrid(w io.Writer, names []string, width int) {
	if len(names) == 0 {
		return
	}
	// 1. Determine maximum name length (add 2 spaces for padding)
	maxLen := 0
	for _, name := range names {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}
	colWidth := maxLen + 2 // 2 spaces padding

	// 2. Determine number of columns
	// Avoid division by zero
	if colWidth > width {
		colWidth = width
	}
	if colWidth == 0 {
		colWidth = 1
	}

	numCols := width / colWidth
	if numCols == 0 {
		numCols = 1
	}
	if numCols > len(names) {
		numCols = len(names)
	}

	// 3. Determine number of rows
	// ceil(len(names) / numCols)
	numRows := (len(names) + numCols - 1) / numCols

	// Pre-compute format string as colWidth is constant for all entries
	format := fmt.Sprintf("%%-%ds", colWidth)

	// 4. Print in column-major order
	// row 0:  idx 0,                idx 0+rows,            idx 0+2*rows...
	// row r:  idx r,                idx r+rows,            ...
	for r := 0; r < numRows; r++ {
		for c := 0; c < numCols; c++ {
			idx := c*numRows + r
			if idx < len(names) {
				_, _ = fmt.Fprintf(w, format, names[idx])
			}
		}
		_, _ = fmt.Fprintln(w)
	}
}

// For testing purposes.
var IsTerminalFunc = func(fd int) bool {
	return term.IsTerminal(fd)
}

var GetTermSizeFunc = func(fd int) (width, height int, err error) {
	return term.GetSize(fd)
}
