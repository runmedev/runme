package testdata

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
	"os"
)

//go:embed users_1m.json.gzip
var Users1MGzip []byte

func UngzipToFile(data []byte, file string) (n int64, err error) {
	reader := bytes.NewReader(data)
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return
	}
	defer func() {
		cerr := gzipReader.Close()
		if err == nil {
			err = cerr
		}
	}()
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()
	return io.Copy(f, gzipReader)
}

// Share testdata across v1 and v2 implementations.
var DetectProgramPathTestCases = []struct {
	Name        string
	ProgramName string
	LanguageID  string
	// Note: Expected paths may vary based on system, so we'll check for common patterns
	ExpectedProgramPathPattern string
	ExpectedArgs               []string
	ExpectError                bool
}{
	{
		Name:                       "bash with empty program name",
		ProgramName:                "",
		LanguageID:                 "bash",
		ExpectedProgramPathPattern: "/bin/bash",
		ExpectedArgs:               []string(nil),
		ExpectError:                false,
	},
	{
		Name:                       "sh with empty program name",
		ProgramName:                "",
		LanguageID:                 "sh",
		ExpectedProgramPathPattern: "/bin/bash", // sh maps to bash in the programByLanguageID
		ExpectedArgs:               []string(nil),
		ExpectError:                false,
	},
	{
		Name:                       "sql with empty program name",
		ProgramName:                "",
		LanguageID:                 "sql",
		ExpectedProgramPathPattern: "/bin/cat",
		ExpectedArgs:               []string(nil),
		ExpectError:                false,
	},
	{
		Name:                       "yaml with empty program name",
		ProgramName:                "",
		LanguageID:                 "yaml",
		ExpectedProgramPathPattern: "/bin/cat",
		ExpectedArgs:               []string(nil),
		ExpectError:                false,
	},
	{
		Name:                       "python with empty program name",
		ProgramName:                "",
		LanguageID:                 "python",
		ExpectedProgramPathPattern: "python", // Should be python3 or python
		ExpectedArgs:               []string(nil),
		ExpectError:                false,
	},
	{
		Name:                       "python with explicit python3 program name",
		ProgramName:                "/usr/bin/python3", // explicitly set python3
		LanguageID:                 "python",
		ExpectedProgramPathPattern: "/usr/bin/python3",
		ExpectedArgs:               []string(nil),
		ExpectError:                false,
	},
	{
		Name:                       "javascript with empty program name",
		ProgramName:                "",
		LanguageID:                 "javascript",
		ExpectedProgramPathPattern: "node",
		ExpectedArgs:               []string(nil),
		ExpectError:                false,
	},
	{
		Name:                       "typescript with empty program name",
		ProgramName:                "",
		LanguageID:                 "typescript",
		ExpectedProgramPathPattern: "deno", // deno run is found instead of ts-node
		ExpectedArgs:               []string{"run"},
		ExpectError:                false,
	},
	{
		Name:                       "ruby with empty program name",
		ProgramName:                "",
		LanguageID:                 "ruby",
		ExpectedProgramPathPattern: "ruby",
		ExpectedArgs:               []string(nil),
		ExpectError:                false,
	},
	{
		Name:                       "bash with explicit program name",
		ProgramName:                "bash",
		LanguageID:                 "bash",
		ExpectedProgramPathPattern: "/bin/bash",
		ExpectedArgs:               []string(nil),
		ExpectError:                false,
	},
	{
		Name:                       "bash with program name and args",
		ProgramName:                "bash -x",
		LanguageID:                 "bash",
		ExpectedProgramPathPattern: "/bin/bash",
		ExpectedArgs:               []string{"-x"},
		ExpectError:                false,
	},
	{
		Name:                       "invalid language ID",
		ProgramName:                "",
		LanguageID:                 "invalid-lang",
		ExpectedProgramPathPattern: "cat", // Falls back to cat when language is invalid
		ExpectedArgs:               []string(nil),
		ExpectError:                false,
	},
	{
		Name:                       "invalid program name",
		ProgramName:                "nonexistent-program-xyz",
		LanguageID:                 "bash",
		ExpectedProgramPathPattern: "",
		ExpectedArgs:               []string(nil),
		ExpectError:                true,
	},
}
