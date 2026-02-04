package safearena

import (
	"fmt"
	"runtime"
	"strings"
)

// stackInfo captures a stack trace for debugging
type stackInfo struct {
	file string
	line int
	fn   string
}

// captureStack captures the current stack location (2 frames up)
func captureStack(skip int) *stackInfo {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return nil
	}

	fn := runtime.FuncForPC(pc)
	fnName := "unknown"
	if fn != nil {
		fnName = fn.Name()
		// Simplify function name
		if idx := strings.LastIndex(fnName, "/"); idx >= 0 {
			fnName = fnName[idx+1:]
		}
	}

	// Simplify file path
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		file = file[idx+1:]
	}

	return &stackInfo{
		file: file,
		line: line,
		fn:   fnName,
	}
}

// errorWithHint creates a panic message with helpful hints
func errorWithHint(arenaID uint64, errorType string, stack *stackInfo, hint string) string {
	var msg strings.Builder

	// Main error
	fmt.Fprintf(&msg, "arena %d: %s", arenaID, errorType)

	// Location
	if stack != nil {
		fmt.Fprintf(&msg, "\n  at %s:%d (%s)", stack.file, stack.line, stack.fn)
	}

	// Hint
	if hint != "" {
		fmt.Fprintf(&msg, "\n\n  💡 Hint: %s", hint)
	}

	return msg.String()
}

// Common hints
const (
	hintUseAfterFree = "Arena was freed before this access. Use Clone() to copy values to heap, or ensure arena lifetime covers all uses."
	hintDoubleFree   = "Arena.Free() was called twice. Make sure Free() is only called once, typically with defer."
	hintAllocAfterFree = "Cannot allocate in a freed arena. Create a new arena or ensure this code runs before Free()."
)
