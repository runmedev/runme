package notebook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"go.uber.org/zap"

	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
	"github.com/runmedev/runme/v3/document"
	"github.com/runmedev/runme/v3/document/editor/editorservice"
	"github.com/runmedev/runme/v3/notebook/daggershell"
)

var validShellFunctionNameRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type NotebookResolver struct {
	notebook *parserv1.Notebook
	editor   parserv1.ParserServiceServer
}

type Option func(*NotebookResolver) error

func WithNotebook(notebook *parserv1.Notebook) Option {
	return func(r *NotebookResolver) error {
		r.notebook = notebook
		return nil
	}
}

func WithSource(source []byte) Option {
	return func(r *NotebookResolver) error {
		des, err := r.editor.Deserialize(context.Background(), &parserv1.DeserializeRequest{Source: source})
		if err != nil {
			return err
		}

		r.notebook = des.Notebook
		return nil
	}
}

func WithDocumentPath(path string) Option {
	return func(r *NotebookResolver) error {
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return WithSource(source)(r)
	}
}

func NewResolver(opts ...Option) (*NotebookResolver, error) {
	r := &NotebookResolver{
		editor: editorservice.NewParserServiceServer(zap.NewNop()),
	}

	// apply options
	for _, opt := range opts {
		err := opt(r)
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}

func (r *NotebookResolver) parseNotebook(context context.Context) (*parserv1.Notebook, error) {
	// make id sticky only for resolving purposes
	for _, cell := range r.notebook.Cells {
		if cell.GetKind() != parserv1.CellKind_CELL_KIND_CODE {
			continue
		}

		if _, ok := cell.Metadata["id"]; ok {
			continue
		}

		if _, ok := cell.Metadata["runme.dev/id"]; !ok {
			continue
		}

		cell.Metadata["id"] = cell.Metadata["runme.dev/id"]
	}

	// properly parse frontmatter and notebook/cell metadata
	s, err := r.editor.Serialize(context, &parserv1.SerializeRequest{Notebook: r.notebook})
	if err != nil {
		return nil, err
	}
	d, err := r.editor.Deserialize(context, &parserv1.DeserializeRequest{Source: s.Result})
	if err != nil {
		return nil, err
	}

	return d.Notebook, nil
}

func GetCellInterpreter(notebook *parserv1.Notebook, cell *parserv1.Cell) string {
	if cell == nil {
		return ""
	}
	interpreter := cell.Metadata["interpreter"]
	if interpreter == "" && notebook != nil && notebook.GetFrontmatter() != nil {
		interpreter = notebook.GetFrontmatter().GetShell()
	}
	return strings.TrimSpace(interpreter)
}

func (r *NotebookResolver) IsDaggerShellCell(notebook *parserv1.Notebook, cell *parserv1.Cell) bool {
	// Use helper to get interpreter
	interpreter := GetCellInterpreter(notebook, cell)
	return strings.Contains(interpreter, "dagger shell")
}

func (r *NotebookResolver) ResolveDaggerShell(context context.Context, cellIndex uint32) (string, error) {
	notebook, err := r.parseNotebook(context)
	if err != nil {
		return "", err
	}

	var targetCell *parserv1.Cell
	targetName := ""
	if int(cellIndex) < 0 || int(cellIndex) >= len(notebook.Cells) {
		return "", fmt.Errorf("cell index out of range")
	}

	cell := notebook.Cells[cellIndex]
	id, okID := cell.Metadata["runme.dev/id"]
	known, okKnown := cell.Metadata["name"]
	generated := cell.Metadata["runme.dev/nameGenerated"]
	if !okID && !okKnown {
		return "", fmt.Errorf("cell metadata is missing required fields")
	}

	isGenerated, err := strconv.ParseBool(generated)
	if !okKnown || isGenerated || err != nil {
		known = fmt.Sprintf("DAGGER_%s", id)
	}

	targetCell = cell
	targetName = known

	// Return cell value directly if target cell is not dagger shell
	if !r.IsDaggerShellCell(notebook, targetCell) {
		return targetCell.GetValue(), nil
	}

	script := daggershell.NewScript()
	for _, cell := range notebook.Cells {
		if cell.GetKind() != parserv1.CellKind_CELL_KIND_CODE {
			continue
		}

		if !r.IsDaggerShellCell(notebook, cell) {
			continue
		}

		languageID := cell.GetLanguageId()
		if languageID != "sh" && languageID != "dagger" {
			continue
		}

		id, okID := cell.Metadata["runme.dev/id"]
		known, okName := cell.Metadata["runme.dev/name"]
		generated := cell.Metadata["runme.dev/nameGenerated"]
		if !okID && !okName {
			continue
		}

		isGenerated, err := strconv.ParseBool(generated)
		if !okName || isGenerated || err != nil {
			known = fmt.Sprintf("DAGGER_%s", id)
		}

		if !isValidShellFunctionName(known) {
			return "", fmt.Errorf("dagger shell integration requires cell name to be a valid shell function name, got %s", known)
		}

		snippet := cell.GetValue()
		if err := script.DefineFunc(known, snippet); err != nil {
			return "", err
		}
	}

	var rendered bytes.Buffer
	shebang := GetCellInterpreter(notebook, targetCell)
	if err := script.RenderWithTarget(&rendered, shebang, targetName); err != nil {
		return "", err
	}

	return rendered.String(), nil
}

func (r *NotebookResolver) GetCellIndexByBlock(block *document.CodeBlock) (uint32, error) {
	return getCellIndexByBlock(r.notebook, block)
}

// todo(sebastian): there are better ways
func getCellIndexByBlock(notebook *parserv1.Notebook, block *document.CodeBlock) (blockIndex uint32, err error) {
	blockIndex = 0
	for i, cell := range notebook.Cells {
		blockValue := string(block.Content())
		if cell.Value == blockValue {
			blockIndex = uint32(i)
			return
		}
	}

	return blockIndex, errors.New("cell for block not found")
}

func isValidShellFunctionName(name string) bool {
	return validShellFunctionNameRe.MatchString(name)
}
