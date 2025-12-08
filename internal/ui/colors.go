package ui

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// ANSI color codes
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	Bold   = "\033[1m"
)

// isTTY checks if stdout is a terminal
func isTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// colorize applies color only if output is a TTY
func colorize(color, msg string) string {
	if !isTTY() {
		return msg
	}
	return color + msg + Reset
}

// OK formats a success message with [OK] prefix in green
func OK(msg string) string {
	prefix := colorize(Green, "[OK]")
	return fmt.Sprintf("%s %s", prefix, msg)
}

// Error formats an error message with [ERROR] prefix in red
func Error(msg string) string {
	prefix := colorize(Red, "[ERROR]")
	return fmt.Sprintf("%s %s", prefix, msg)
}

// Warn formats a warning message with [WARN] prefix in yellow
func Warn(msg string) string {
	prefix := colorize(Yellow, "[WARN]")
	return fmt.Sprintf("%s %s", prefix, msg)
}

// Info formats an info message with [INFO] prefix in blue
func Info(msg string) string {
	prefix := colorize(Blue, "[INFO]")
	return fmt.Sprintf("%s %s", prefix, msg)
}

// Title formats a section title in bold cyan
func Title(msg string) string {
	prefix := colorize(Bold+Cyan, fmt.Sprintf("[%s]", msg))
	return prefix
}

// TitleWithDesc formats a section title with description
func TitleWithDesc(title, desc string) string {
	prefix := colorize(Bold+Cyan, fmt.Sprintf("[%s]", title))
	return fmt.Sprintf("%s %s", prefix, desc)
}

// Done formats a completion message with [DONE] prefix in green
func Done(msg string) string {
	prefix := colorize(Green+Bold, "[DONE]")
	return fmt.Sprintf("%s %s", prefix, msg)
}

// PrintOK prints a success message
func PrintOK(msg string) {
	fmt.Println(OK(msg))
}

// PrintError prints an error message
func PrintError(msg string) {
	fmt.Println(Error(msg))
}

// PrintWarn prints a warning message
func PrintWarn(msg string) {
	fmt.Println(Warn(msg))
}

// PrintInfo prints an info message
func PrintInfo(msg string) {
	fmt.Println(Info(msg))
}

// PrintTitle prints a section title
func PrintTitle(title, desc string) {
	fmt.Println(TitleWithDesc(title, desc))
}

// PrintDone prints a completion message
func PrintDone(msg string) {
	fmt.Println(Done(msg))
}

// Indent returns the message with indentation
func Indent(msg string) string {
	return "     " + msg
}

// PrintIndent prints an indented message
func PrintIndent(msg string) {
	fmt.Println(Indent(msg))
}
