package display

import (
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/ioluas/gopherutils/internal/fsutil"
	lsconfig "github.com/ioluas/gopherutils/utils/file/ls/internal/config"
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

func escapeRuneC(r rune) string {
	switch r {
	case '\a':
		return `\a`
	case '\b':
		return `\b`
	case '\f':
		return `\f`
	case '\n':
		return `\n`
	case '\r':
		return `\r`
	case '\t':
		return `\t`
	case '\v':
		return `\v`
	case '\\':
		return `\\`
	case '\'':
		return `\'`
	case '"':
		return `\"`
	default:
		if r <= 0xFF {
			return fmt.Sprintf("\\x%02X", r)
		}
		if r <= 0xFFFF {
			return fmt.Sprintf("\\u%04X", r)
		}
		return fmt.Sprintf("\\U%08X", r)
	}
}

func quoteLocale(name string) string {
	var b strings.Builder
	for _, r := range name {
		if unicode.IsPrint(r) && r != '\u0000' {
			b.WriteRune(r)
			continue
		}
		b.WriteString(escapeRuneC(r))
	}
	return "\u2018" + b.String() + "\u2019"
}

// ShellQuote quotes a filename for shell usage.
// It delegates to quoteShell(name, false), wrapping the name in single quotes if needed.
// Embedded single quotes are escaped as '\” (close quote, literal '\', reopen quote).
func ShellQuote(name string) string {
	return quoteShell(name, false)
}

func quoteShellANSI(name string, always bool) string {
	needsQuote := false
	if !always {
		for _, r := range name {
			// Characters that usually require quoting in shell
			if r == ' ' || r == '\t' || r == '\n' || r == '\'' || r == '"' ||
				r == '\\' || r == '$' || r == '`' || r == '(' || r == ')' ||
				r == '<' || r == '>' || r == '|' || r == '&' || r == ';' || r == '*' ||
				r == '?' || r == '[' || r == ']' || r == '#' || r == '~' {
				needsQuote = true
				break
			}
			// Also quote control characters (though usually ls -b changes them)
			if r < 32 || r == 127 {
				needsQuote = true
				break
			}
		}
	}

	if !always && !needsQuote && len(name) > 0 {
		return name
	}

	var b strings.Builder
	b.WriteString("$'")
	for _, r := range name {
		if unicode.IsPrint(r) && r != '\u0000' && r != '\'' && r != '\\' {
			b.WriteRune(r)
			continue
		}
		b.WriteString(escapeRuneC(r))
	}
	b.WriteByte('\'')
	return b.String()
}

func quoteShell(name string, always bool) string {
	needsQuote := false
	if !always {
		for _, r := range name {
			// Characters that usually require quoting in shell
			if r == ' ' || r == '\t' || r == '\n' || r == '\'' || r == '"' ||
				r == '\\' || r == '$' || r == '`' || r == '(' || r == ')' ||
				r == '<' || r == '>' || r == '|' || r == '&' || r == ';' || r == '*' ||
				r == '?' || r == '[' || r == ']' || r == '#' || r == '~' {
				needsQuote = true
				break
			}
			// Also quote control characters (though usually ls -b changes them)
			if r < 32 || r == 127 {
				needsQuote = true
				break
			}
		}
	}

	if !always && !needsQuote && len(name) > 0 {
		return name
	}

	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range name {
		if r == '\'' {
			b.WriteString(`'\''`)
		} else {
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}

// Quote quotes the name according to the configured style.
func Quote(name string, style lsconfig.QuotingStyle) string {
	switch style {
	case lsconfig.QuotingStyleEscape:
		return QuoteName(name)
	case lsconfig.QuotingStyleLocale:
		return quoteLocale(name)
	case lsconfig.QuotingStyleShell:
		return quoteShell(name, false)
	case lsconfig.QuotingStyleShellAlways:
		return quoteShell(name, true)
	case lsconfig.QuotingStyleShellEscape:
		return quoteShellANSI(name, false)
	case lsconfig.QuotingStyleShellEscapeAlways:
		return quoteShellANSI(name, true)
	case lsconfig.QuotingStyleC:
		escaped := strings.ReplaceAll(QuoteName(name), `"`, `\"`)
		return `"` + escaped + `"`
	default:
		return name
	}
}

func PrintLongList(stdout, stderr io.Writer, entries []os.DirEntry, config *lsconfig.Config, printTotal bool) bool {
	if len(entries) == 0 {
		return false
	}

	var hadError bool
	effectiveStyle := config.QuotingStyle
	if effectiveStyle == lsconfig.QuotingStyleLiteral && config.Escape {
		effectiveStyle = lsconfig.QuotingStyleEscape
	}

	// Gather all file info
	details := make([]fileDetails, 0, len(entries))
	var maxLinkLen, maxOwnerLen, maxAuthorLen, maxGroupLen, maxSizeLen int
	var totalBlocks int64

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
			totalBlocks += sysStat.Blocks
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
		name = Quote(name, effectiveStyle)

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
	var bc *ByteCountingWriter
	var offsets []int64

	if config.Dired {
		bc = &ByteCountingWriter{Writer: stdout}
		stdout = bc
	}

	if printTotal {
		// Output block count in 1K units (standard ls behavior)
		prefix := ""
		if config.Dired {
			prefix = "  "
		}
		_, _ = fmt.Fprintf(stdout, "%stotal %d\n", prefix, (totalBlocks+1)/2)
	}

	for _, d := range details {
		format, args := longListFormatArgs(d, widths, config)
		if config.Dired {
			format = "  " + format
		}
		_, _ = fmt.Fprintf(stdout, format, args...)

		if config.Dired {
			offsets = append(offsets, bc.Count)
		}
		_, _ = fmt.Fprint(stdout, d.name)
		if config.Dired {
			offsets = append(offsets, bc.Count)
		}
		_, _ = fmt.Fprintln(stdout)
	}

	if config.Dired && len(offsets) > 0 {
		_, _ = fmt.Fprint(stdout, "//DIRED//")
		for _, off := range offsets {
			_, _ = fmt.Fprintf(stdout, " %d", off)
		}
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintf(stdout, "//DIRED-OPTIONS// --quoting-style=%s\n", effectiveStyle.String())
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

func longListFormatArgs(d fileDetails, widths longListWidths, config *lsconfig.Config) (string, []interface{}) {
	timeStr := timeutil.FormatTime(d.modTime, config)
	switch {
	case config.ShowAuthor && config.NoGroup:
		return "%s %*d %-*s %-*s %*s %s ", []interface{}{
			d.mode,
			widths.link, d.nlink,
			widths.owner, d.owner,
			widths.author, d.author,
			widths.size, d.sizeStr,
			timeStr,
		}
	case config.ShowAuthor:
		return "%s %*d %-*s %-*s %-*s %*s %s ", []interface{}{
			d.mode,
			widths.link, d.nlink,
			widths.owner, d.owner,
			widths.author, d.author,
			widths.group, d.group,
			widths.size, d.sizeStr,
			timeStr,
		}
	case config.NoGroup:
		return "%s %*d %-*s %*s %s ", []interface{}{
			d.mode,
			widths.link, d.nlink,
			widths.owner, d.owner,
			widths.size, d.sizeStr,
			timeStr,
		}
	default:
		// Print without author column.
		return "%s %*d %-*s %-*s %*s %s ", []interface{}{
			d.mode,
			widths.link, d.nlink,
			widths.owner, d.owner,
			widths.group, d.group,
			widths.size, d.sizeStr,
			timeStr,
		}
	}
}

// printEntries prints entry names to the output writer.
// It detects if the writer is a terminal to format output in columns.
func PrintEntries(w io.Writer, names []string, config *lsconfig.Config) {
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

	// Determine output format based on FormatMode (last flag wins)
	useGrid := false
	switch {
	case config != nil && config.FormatMode == lsconfig.FormatColumnate:
		// -C was explicitly specified last
		useGrid = true
	case config != nil && config.FormatMode == lsconfig.FormatOnePerLine:
		// -1 was explicitly specified last
		useGrid = false
	case isTerminal:
		// No explicit format flag, default to grid on terminal
		useGrid = true
	default:
		// No explicit format flag, not a terminal
		useGrid = false
	}

	if useGrid {
		width := termWidth
		if !isTerminal {
			width = 80
		}
		PrintGrid(w, names, width)
	} else {
		// Not columnated, or not a terminal: one entry per line
		for _, name := range names {
			_, _ = fmt.Fprintln(w, name)
		}
	}
}

// printGrid formats names into a grid that fits within width.
// Users column-major order (standard ls behavior).
func PrintGrid(w io.Writer, names []string, termWidth int) {
	if len(names) == 0 {
		return
	}

	n := len(names)
	padding := 2

	// Try with increasing number of rows
	for rows := 1; rows <= n; rows++ {
		cols := (n + rows - 1) / rows

		// Calculate width of each column
		colWidths := make([]int, cols)
		for c := 0; c < cols; c++ {
			for r := 0; r < rows; r++ {
				idx := c*rows + r
				if idx < n {
					l := len(names[idx])
					if l > colWidths[c] {
						colWidths[c] = l
					}
				}
			}
		}

		// Calculate total width required
		totalWidth := 0
		for _, width := range colWidths {
			totalWidth += width
		}
		totalWidth += (cols - 1) * padding

		// If it fits, or if we are forced to 1 column (rows==n), print it
		if totalWidth <= termWidth || rows == n {
			for r := 0; r < rows; r++ {
				for c := 0; c < cols; c++ {
					idx := c*rows + r
					if idx < n {
						// padding only between columns
						// Is this the last column?
						// Note: The totalWidth check assumes all columns present.
						// When printing, we print logical columns.

						// Standard ls behavior: pad to column width, except the last column doesn't arguably need padding if it's rightmost?
						// Actually ls usually pads all except maybe the very last visible one on the line.
						// To be safe and consistent with format strings:
						// printf "%-*s  " width name

						if c < cols-1 {
							// Not the last column, print with padding
							format := fmt.Sprintf("%%-%ds", colWidths[c]+padding)
							_, _ = fmt.Fprintf(w, format, names[idx])
						} else {
							// Last column, just print the name
							_, _ = fmt.Fprint(w, names[idx])
						}
					}
				}
				_, _ = fmt.Fprintln(w)
			}
			return
		}
	}
}

// For testing purposes.
var IsTerminalFunc = func(fd int) bool {
	return term.IsTerminal(fd)
}

var GetTermSizeFunc = func(fd int) (width, height int, err error) {
	return term.GetSize(fd)
}

// ByteCountingWriter counts bytes written to the underlying writer
type ByteCountingWriter struct {
	io.Writer
	Count int64
}

func (w *ByteCountingWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	w.Count += int64(n)
	return n, err
}
