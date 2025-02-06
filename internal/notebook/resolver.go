package notebook

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/stateful/runme/v3/internal/notebook/daggershell"
	parserv1 "github.com/stateful/runme/v3/pkg/api/gen/proto/go/runme/parser/v1"
	"github.com/stateful/runme/v3/pkg/document/editor/editorservice"
	"go.uber.org/zap"
)

type NotebookResolver struct {
	notebook *parserv1.Notebook
	editor   parserv1.ParserServiceServer
}

func NewNotebookResolver(notebook *parserv1.Notebook) *NotebookResolver {
	return &NotebookResolver{
		notebook: notebook,
		editor:   editorservice.NewParserServiceServer(zap.NewNop()),
	}
}

func (r *NotebookResolver) parseNotebook(context context.Context) (*parserv1.Notebook, error) {
	// make id sticky only for resolving purposes
	for _, cell := range r.notebook.Cells {
		if cell.GetKind() != parserv1.CellKind_CELL_KIND_CODE {
			continue
		}

		_, ok := cell.Metadata["id"]
		if ok {
			continue
		}

		if cell.Metadata == nil {
			return nil, fmt.Errorf("cell metadata is missing")
		}

		cell.Metadata["id"] = cell.Metadata["runme.dev/id"]
	}

	// properly parse frontmatter and notebook/cell metadata
	ser, err := r.editor.Serialize(context, &parserv1.SerializeRequest{Notebook: r.notebook})
	if err != nil {
		return nil, err
	}
	des, err := r.editor.Deserialize(context, &parserv1.DeserializeRequest{Source: ser.Result})
	if err != nil {
		return nil, err
	}

	return des.Notebook, nil
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

	if notebook.Frontmatter == nil || !strings.Contains(strings.Trim(notebook.Frontmatter.Shell, " \t\r\n"), "dagger shell") {
		return targetCell.GetValue(), nil
	}

	script := daggershell.NewScript()
	for _, cell := range notebook.Cells {
		if cell.GetKind() != parserv1.CellKind_CELL_KIND_CODE {
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

		snippet := cell.GetValue()
		if err := script.DeclareFunc(known, snippet); err != nil {
			return "", err
		}
	}

	var rendered bytes.Buffer
	if err := script.RenderWithCall(&rendered, targetName); err != nil {
		return "", err
	}

	return rendered.String(), nil
}
