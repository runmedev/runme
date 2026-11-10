package execution

import "strings"

// Intent describes the program requested by a caller before executable lookup
// and runtime-specific resolution.
type Intent struct {
	programName string
	arguments   []string
}

// NewIntent returns an intent that owns a copy of arguments.
func NewIntent(programName string, arguments []string) Intent {
	return Intent{programName: programName, arguments: clone(arguments)}
}

// ProgramName returns the requested program name.
func (i Intent) ProgramName() string {
	return i.programName
}

// Arguments returns a copy of the requested arguments.
func (i Intent) Arguments() []string {
	return clone(i.arguments)
}

// Invocation describes a resolved executable and its arguments.
type Invocation struct {
	executable string
	arguments  []string
}

// NewInvocation returns an invocation that owns a copy of arguments.
func NewInvocation(executable string, arguments []string) Invocation {
	return Invocation{executable: executable, arguments: clone(arguments)}
}

// Executable returns the resolved executable.
func (i Invocation) Executable() string {
	return i.executable
}

// Arguments returns a copy of the resolved arguments.
func (i Invocation) Arguments() []string {
	return clone(i.arguments)
}

// Fields returns the executable followed by a copy of its arguments.
func (i Invocation) Fields() []string {
	return append([]string{i.executable}, i.arguments...)
}

// String returns the invocation as space-separated fields for diagnostics.
func (i Invocation) String() string {
	return strings.Join(i.Fields(), " ")
}

// WithArguments returns a copy with arguments appended.
func (i Invocation) WithArguments(arguments ...string) Invocation {
	result := clone(i.arguments)
	result = append(result, arguments...)
	return NewInvocation(i.executable, result)
}

func clone(values []string) []string {
	return append([]string(nil), values...)
}
