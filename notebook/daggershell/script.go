package daggershell

import (
	"errors"
	"io"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

type Script struct {
	stmts   []*syntax.Stmt
	printer *syntax.Printer
	parser  *syntax.Parser
}

func NewScript() *Script {
	return &Script{
		stmts: []*syntax.Stmt{},
		printer: syntax.NewPrinter(
			syntax.Indent(2),
			syntax.BinaryNextLine(true),
			syntax.FunctionNextLine(true),
		),
		parser: syntax.NewParser(),
	}
}

func (s *Script) DefineFunc(name, body string) error {
	stmt := &syntax.Stmt{
		Cmd: &syntax.FuncDecl{
			Parens: true,
			Name: &syntax.Lit{
				Value: name,
			},
			Body: &syntax.Stmt{
				Cmd: &syntax.Block{},
			},
		},
	}

	snippet, err := s.parser.Parse(strings.NewReader(body), name)
	if err != nil {
		return err
	}

	// assign stmts to insert function body
	syntax.Walk(stmt, func(node syntax.Node) bool {
		if block, ok := node.(*syntax.Block); ok {
			block.Stmts = snippet.Stmts
			return false
		}

		// todo(sebastian): check for validity, e.g. func def inside itself
		return true
	})

	s.stmts = append(s.stmts, stmt)

	return nil
}

// Render renders the script with a required shebang (e.g. "/bin/bash").
func (s *Script) Render(w io.Writer, shebang string) error {
	return s.render(w, shebang, "")
}

// RenderWithTarget renders the script, calling the specified function and adding a required shebang.
func (s *Script) RenderWithTarget(w io.Writer, shebang, target string) error {
	return s.render(w, shebang, target)
}

// render renders the script, optionally calling a function and adding a shebang.
func (s *Script) render(w io.Writer, shebang, target string) error {
	if parts := strings.Split(shebang, " "); len(parts) >= 2 {
		shebang = "/usr/bin/env " + shebang
	}

	var file *syntax.File

	// Prepare the statements, potentially adding the function call
	var stmts []*syntax.Stmt

	// Handle shebang first if provided
	if shebang != "" {
		// Create shebang as the first line
		shebangStmt := &syntax.Stmt{
			Comments: []syntax.Comment{
				{
					Hash: syntax.Pos{},
					Text: "!" + shebang, // e.g. "!/bin/bash"
				},
			},
		}

		// Start with the shebang
		stmts = append(stmts, shebangStmt)
	}

	// Handle the case where no target function is specified
	if target == "" {
		// Just add all statements after potential shebang
		stmts = append(stmts, s.stmts...)
		file = &syntax.File{
			Name:  "DaggerShellScript",
			Stmts: stmts,
		}
		return s.printer.Print(w, file)
	}

	// Copy the original statements
	funcStmts := make([]*syntax.Stmt, len(s.stmts))
	copy(funcStmts, s.stmts)

	// Add function statements after potential shebang
	stmts = append(stmts, funcStmts...)

	file = &syntax.File{
		Name:  "DaggerShellScript",
		Stmts: stmts,
	}

	validFuncName := false
	// check if func name was previously declared
	syntax.Walk(file, func(node syntax.Node) bool {
		decl, ok := node.(*syntax.FuncDecl)
		if !ok {
			return true
		}

		if decl.Name.Value == target {
			validFuncName = true
			return false
		}

		return true
	})

	if !validFuncName {
		return errors.New("undeclared function name")
	}

	// Add the function call at the end
	file.Stmts = append(file.Stmts, &syntax.Stmt{
		Cmd: &syntax.CallExpr{
			Args: []*syntax.Word{
				{
					Parts: []syntax.WordPart{
						&syntax.Lit{Value: target},
					},
				},
			},
		},
	})

	return s.printer.Print(w, file)
}
