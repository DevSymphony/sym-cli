package cmd

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// ANSI color codes
const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	bold   = "\033[1m"
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
	return color + msg + reset
}

// ok formats a success message with [OK] prefix in green
func ok(msg string) string {
	prefix := colorize(green, "[OK]")
	return fmt.Sprintf("%s %s", prefix, msg)
}

// formatError formats an error message with [ERROR] prefix in red
func formatError(msg string) string {
	prefix := colorize(red, "[ERROR]")
	return fmt.Sprintf("%s %s", prefix, msg)
}

// warn formats a warning message with [WARN] prefix in yellow
func warn(msg string) string {
	prefix := colorize(yellow, "[WARN]")
	return fmt.Sprintf("%s %s", prefix, msg)
}

// titleWithDesc formats a section title with description
func titleWithDesc(title, desc string) string {
	prefix := colorize(bold+cyan, fmt.Sprintf("[%s]", title))
	return fmt.Sprintf("%s %s", prefix, desc)
}

// done formats a completion message with [DONE] prefix in green
func done(msg string) string {
	prefix := colorize(green+bold, "[DONE]")
	return fmt.Sprintf("%s %s", prefix, msg)
}

// printOK prints a success message
func printOK(msg string) {
	fmt.Println(ok(msg))
}

// printError prints an error message
func printError(msg string) {
	fmt.Println(formatError(msg))
}

// printWarn prints a warning message
func printWarn(msg string) {
	fmt.Println(warn(msg))
}

// printTitle prints a section title
func printTitle(title, desc string) {
	fmt.Println(titleWithDesc(title, desc))
}

// printDone prints a completion message
func printDone(msg string) {
	fmt.Println(done(msg))
}

// indent returns the message with indentation
func indent(msg string) string {
	return "     " + msg
}
