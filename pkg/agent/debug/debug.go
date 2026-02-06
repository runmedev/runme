package debug

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// ThisCaller returns a string identifying the function that invoked the function where ThisCaller was invoked.
// The return value is ${package}/${file}:${linenumber}.
// For example:
// Suppose we have file: mygomod/pkg/parent.go
//
//	func parent() {
//	  child()
//	}
//
// file: mygomod/pkg/child.go
//
//	func child() {
//	 c := debug.ThisCaller()
//	}
//
// The return value of ThisCaller will be "pkg/parent.go:2"
// Returning the full path wouldn't be useful because it would be the full path on the machine where the code
// was compiled
func ThisCaller() string {
	// We need to skip 2 frames; get frame and the function to get the caller of
	frame := getFrame(2)

	packagePath, fName := filepath.Split(frame.File)
	packageName := filepath.Base(packagePath)

	return fmt.Sprintf("%v/%v:%v", packageName, fName, frame.Line)
}

func getFrame(numToSkip int) runtime.Frame {
	// We add 2 because we don't want runtime.Callers and getFrame to be included
	targetIndex := numToSkip + 2

	counters := make([]uintptr, targetIndex+2)

	frame := runtime.Frame{Function: "unknown"}
	i := runtime.Callers(0, counters)

	if i > 0 {
		frames := runtime.CallersFrames(counters[:i])
		for more, fIndex := true, 0; more && fIndex <= targetIndex; fIndex++ {
			var candidate runtime.Frame
			candidate, more = frames.Next()
			if fIndex == targetIndex {
				frame = candidate
			}
		}
	}
	return frame
}
