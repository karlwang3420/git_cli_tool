package log

import (
	"fmt"
	"os"
)

// FormatWarning formats a warning message with a consistent structure
func FormatWarning(message string) string {
	return fmt.Sprintf("[WARN]    %s", message)
}

// FormatSuccess formats a success message with a consistent structure
func FormatSuccess(message string) string {
	return fmt.Sprintf("[SUCCESS] %s", message)
}

// FormatDebug formats a debug message with a consistent structure
func FormatDebug(message string) string {
	return fmt.Sprintf("[DEBUG]   %s", message)
}

// PrintError prints an error message with the appropriate error code and exits with code 1
func PrintError(code string, description string, err error) {
	fmt.Fprintln(os.Stderr, FormatError(code, description, err))
	os.Exit(1)
}

// PrintErrorNoExit prints an error message with the appropriate error code without exiting
func PrintErrorNoExit(code string, description string, err error) {
	fmt.Fprintln(os.Stderr, FormatError(code, description, err))
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	fmt.Fprintln(os.Stderr, FormatWarning(message))
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	fmt.Println(FormatSuccess(message))
}

// PrintInfo prints an info message
func PrintInfo(message string) {
	fmt.Println((message))
}

// PrintOperation prints a message about an operation being performed
func PrintOperation(operation string) {
	fmt.Println((operation))
}

// PrintDebug prints a debug message
func PrintDebug(message string) {
	fmt.Fprintln(os.Stderr, FormatDebug(message))
}

// PrintOperationResult prints the result of an operation
func PrintOperationResult(operation string, success bool) {
	if success {
		PrintSuccess(fmt.Sprintf("%s completed successfully", operation))
	} else {
		PrintWarning(fmt.Sprintf("%s completed with errors", operation))
	}
}
