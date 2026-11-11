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

// Render renders the script body.
func (s *Script) Render(w io.Writer) error {
	return s.render(w, "")
}

// RenderWithTarget renders the script body and calls the specified function.
func (s *Script) RenderWithTarget(w io.Writer, target string) error {
	return s.render(w, target)
}

// render renders the script body, optionally calling a function.
func (s *Script) render(w io.Writer, target string) error {
	var file *syntax.File

	// Handle the case where no target function is specified
	if target == "" {
		file = &syntax.File{
			Name:  "DaggerShellScript",
			Stmts: s.stmts,
		}
		return s.printer.Print(w, file)
	}

	// Copy the original statements
	funcStmts := make([]*syntax.Stmt, len(s.stmts))
	copy(funcStmts, s.stmts)

	file = &syntax.File{
		Name:  "DaggerShellScript",
		Stmts: funcStmts,
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
