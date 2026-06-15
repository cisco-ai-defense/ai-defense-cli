package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

// ANSI styles mirroring Typer/Rich's default help colors (what the embedded
// aibom uses) so native and aibom help look identical: bold-cyan option names,
// bold-green short flags, bold-yellow type metavars, faint borders/titles, and a
// bold-yellow "Usage:" label. We emit raw ANSI (gated on the destination stream
// being a TTY) rather than going through a color-profile writer.
const (
	cReset  = "\x1b[0m"
	cBold   = "\x1b[1m"
	cOption = "\x1b[1;36m" // bold cyan
	cSwitch = "\x1b[1;32m" // bold green
	cType   = "\x1b[1;33m" // bold yellow
	cUsage  = "\x1b[1;33m" // bold yellow
	cBorder = "\x1b[2m"    // faint (matches Typer/Rich panel borders)
)

// colorEnabled decides whether to emit ANSI for the given destination writer.
// Color is tied to the stream actually being rendered to (so `--help >file`
// stays plain even if the other standard stream is a TTY), honoring NO_COLOR /
// FORCE_COLOR / CLICOLOR_FORCE.
func colorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("FORCE_COLOR") != "" || os.Getenv("CLICOLOR_FORCE") != "" {
		return true
	}
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

func paint(color bool, code, s string) string {
	if !color || s == "" {
		return s
	}
	return code + s + cReset
}

// panelHelpFunc renders a command's help as titled panels, mirroring the
// embedded aibom (Typer/Rich) help layout so the whole CLI looks uniform.
func panelHelpFunc(cmd *cobra.Command, _ []string) {
	w := cmd.OutOrStdout()
	fmt.Fprint(w, renderHelp(cmd, w))
}

// panelUsageFunc renders usage in the same panel style. It writes to the command
// output writer (not stderr directly) so cobra's UsageString() can capture it —
// e.g. when Run() prints usage after its own error/hint block.
func panelUsageFunc(cmd *cobra.Command) error {
	w := cmd.OutOrStderr()
	fmt.Fprint(w, renderHelp(cmd, w))
	return nil
}

// helpWidth returns the width to render against: the destination terminal's
// width, or 80 when the destination is not a terminal (a pipe/file, or the
// buffer cobra uses for UsageString()).
func helpWidth(w io.Writer) int {
	if f, ok := w.(*os.File); ok {
		if width, _, err := term.GetSize(int(f.Fd())); err == nil && width > 0 {
			return width
		}
	}
	return 80
}

func renderHelp(cmd *cobra.Command, w io.Writer) string {
	color := colorEnabled(w)
	total := helpWidth(w)
	content := total - 4 // "│ " + " │"
	if content < 10 {
		content = 10
	}

	tail := "[OPTIONS]"
	if cmd.HasAvailableSubCommands() {
		tail = "[OPTIONS] COMMAND [ARGS]..."
	}

	var b strings.Builder
	b.WriteString("\n " + paint(color, cUsage, "Usage:") + " " +
		paint(color, cBold, cmd.CommandPath()+" "+tail) + "\n")

	desc := cmd.Long
	if desc == "" || cmd.HasAvailableSubCommands() {
		desc = cmd.Short
	}
	if desc != "" {
		b.WriteString("\n")
		for _, l := range wrapText(desc, total-1) {
			b.WriteString(" " + l + "\n")
		}
	}

	if body := commandsBody(color, cmd, content); body != "" {
		b.WriteString("\n" + panel(color, "Commands", body, total) + "\n")
	}
	if body := flagsBody(color, cmd.LocalFlags(), content); body != "" {
		b.WriteString("\n" + panel(color, "Options", body, total) + "\n")
	}
	if body := flagsBody(color, cmd.InheritedFlags(), content); body != "" {
		b.WriteString("\n" + panel(color, "Global Options", body, total) + "\n")
	}
	return b.String()
}

// panel draws body inside a rounded, titled box of the given total width, with
// the title spliced into the top border (╭─ Title ──...──╮) like Rich panels.
func panel(color bool, title, body string, total int) string {
	inner := total - 4
	if inner < 4 {
		inner = 4
	}

	var b strings.Builder
	prefix := "╭─ " + title + " "
	fill := total - displayWidth(prefix) - 1 // -1 for ╮
	if fill < 0 {
		fill = 0
	}
	b.WriteString(paint(color, cBorder, prefix+strings.Repeat("─", fill)+"╮") + "\n")

	for _, line := range strings.Split(body, "\n") {
		gap := inner - displayWidth(line)
		if gap < 0 {
			gap = 0
		}
		b.WriteString(paint(color, cBorder, "│") + " " + line +
			strings.Repeat(" ", gap) + " " + paint(color, cBorder, "│") + "\n")
	}

	b.WriteString(paint(color, cBorder, "╰"+strings.Repeat("─", total-2)+"╯"))
	return b.String()
}

func commandsBody(color bool, cmd *cobra.Command, content int) string {
	type row struct{ name, help string }
	var rows []row
	nameW := 0
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		rows = append(rows, row{c.Name(), c.Short})
		nameW = maxi(nameW, len(c.Name()))
	}
	if len(rows) == 0 {
		return ""
	}
	if nameW > 24 {
		nameW = 24
	}
	helpW := content - nameW - 2
	if helpW < 10 {
		helpW = 10
	}

	var b strings.Builder
	for i, r := range rows {
		help := wrapText(r.help, helpW)
		indent := strings.Repeat(" ", nameW+2)
		b.WriteString(pad(paint(color, cOption, r.name), nameW) + "  " + first(help))
		for _, l := range rest(help) {
			b.WriteString("\n" + indent + l)
		}
		if i < len(rows)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func flagsBody(color bool, fs *pflag.FlagSet, content int) string {
	type row struct{ long, short, typ, help string }
	var rows []row
	longW, shortW, typeW := 0, 0, 0
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		long := "--" + f.Name
		short := ""
		if f.Shorthand != "" {
			short = "-" + f.Shorthand
		}
		typ := flagType(f)
		help := strings.TrimSpace(f.Usage + " " + defaultNote(f))
		rows = append(rows, row{long, short, typ, help})
		longW = maxi(longW, len(long))
		shortW = maxi(shortW, len(short))
		typeW = maxi(typeW, len(typ))
	})
	if len(rows) == 0 {
		return ""
	}
	if longW > 28 {
		longW = 28
	}

	// Column layout: --long  -s  TYPE  help (help hang-indented). Skip the short
	// or type columns entirely when no flag in this set uses them.
	prefixW := longW + 2
	if shortW > 0 {
		prefixW += shortW + 2
	}
	if typeW > 0 {
		prefixW += typeW + 2
	}
	helpW := content - prefixW
	if helpW < 10 {
		helpW = 10
	}

	var b strings.Builder
	for i, r := range rows {
		left := pad(paint(color, cOption, r.long), longW) + "  "
		if shortW > 0 {
			left += pad(paint(color, cSwitch, r.short), shortW) + "  "
		}
		if typeW > 0 {
			left += pad(paint(color, cType, r.typ), typeW) + "  "
		}
		indent := strings.Repeat(" ", prefixW)
		help := wrapText(r.help, helpW)
		b.WriteString(left + first(help))
		for _, l := range rest(help) {
			b.WriteString("\n" + indent + l)
		}
		if i < len(rows)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func flagType(f *pflag.Flag) string {
	switch f.Value.Type() {
	case "bool", "count":
		return ""
	case "string":
		return "TEXT"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return "INTEGER"
	case "float32", "float64":
		return "FLOAT"
	case "stringArray", "stringSlice":
		return "TEXT"
	default:
		return strings.ToUpper(f.Value.Type())
	}
}

func defaultNote(f *pflag.Flag) string {
	switch f.Value.Type() {
	case "bool", "count":
		return ""
	}
	if f.DefValue == "" || f.DefValue == "[]" || f.DefValue == "0" {
		return ""
	}
	return "[default: " + f.DefValue + "]"
}

// wrapText word-wraps s to width w, preserving explicit newlines.
func wrapText(s string, w int) []string {
	if w <= 0 {
		return []string{s}
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := words[0]
		for _, word := range words[1:] {
			if displayWidth(line)+1+displayWidth(word) <= w {
				line += " " + word
			} else {
				out = append(out, line)
				line = word
			}
		}
		out = append(out, line)
	}
	return out
}

// pad right-pads s to visible width w (ANSI-aware via displayWidth).
func pad(s string, w int) string {
	if d := w - displayWidth(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}

// displayWidth returns the visible width of s, ignoring ANSI SGR escape
// sequences (e.g. "\x1b[1;36m"). Help text is ASCII + box-drawing glyphs, all
// single-width, so a rune count suffices — no Unicode width tables needed.
func displayWidth(s string) int {
	w, inEsc := 0, false
	for _, r := range s {
		switch {
		case inEsc:
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false // a final byte (m, K, ...) ends the escape
			}
		case r == 0x1b: // ESC
			inEsc = true
		default:
			w++
		}
	}
	return w
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func rest(s []string) []string {
	if len(s) <= 1 {
		return nil
	}
	return s[1:]
}
