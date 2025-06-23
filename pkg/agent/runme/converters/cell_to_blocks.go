package converters

import (
	"github.com/pkg/errors"

	"github.com/runmedev/runme/v3/api/gen/proto/go/agent"
	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
)

// Doc represents a document
// This is a hack for when we copied this code over from foyle; if we keep doc; we should make it a proto.
type Doc struct {
	Blocks []*agent.Block
}

// NotebookToDoc converts a runme Notebook to a foyle Doc
func NotebookToDoc(nb *parserv1.Notebook) (*Doc, error) {
	if nb == nil {
		return nil, errors.New("Notebook is nil")
	}

	doc := &Doc{
		Blocks: make([]*agent.Block, 0, len(nb.Cells)),
	}

	for _, cell := range nb.Cells {
		block, err := CellToBlock(cell)
		if err != nil {
			return nil, err
		}
		doc.Blocks = append(doc.Blocks, block)
	}

	return doc, nil
}

// CellToBlock converts a runme Cell to a foyle Block
//
// N.B. cell metadata is currently ignored.
func CellToBlock(cell *parserv1.Cell) (*agent.Block, error) {
	if cell == nil {
		return nil, errors.New("Cell is nil")
	}

	blockOutputs := make([]*agent.BlockOutput, 0, len(cell.Outputs))

	for _, output := range cell.Outputs {
		bOutput, err := CellOutputToBlockOutput(output)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to convert CellOutput to BlockOutput")
		}
		blockOutputs = append(blockOutputs, bOutput)
	}
	blockKind := CellKindToBlockKind(cell.Kind)

	id := ""
	if cell.Metadata != nil {
		newId := GetCellID(cell)
		if newId != "" {
			id = newId
		}
	}

	return &agent.Block{
		Id:       id,
		Language: cell.LanguageId,
		Contents: cell.Value,
		Kind:     blockKind,
		Outputs:  blockOutputs,
		Metadata: cell.Metadata,
	}, nil
}

// GetCellID returns the ID of a cell if it exists or none if it doesn't
func GetCellID(cell *parserv1.Cell) string {
	if cell.Metadata != nil {
		// See this thread
		// See this thread https://discord.com/channels/1102639988832735374/1218835142962053193/1278863895813165128
		// RunMe uses two different fields for the ID field. We check both because the field we get could depend
		// On how the cell was generated e.g. whether it went through the serializer or not.
		if id, ok := cell.Metadata[RunmeIdField]; ok {
			return id
		}
		if id, ok := cell.Metadata[IdField]; ok {
			return id
		}
	}
	return ""
}

func SetCellID(cell *parserv1.Cell, id string) {
	// Delete any existing IDs
	for _, idField := range []string{IdField, RunmeIdField} {
		delete(cell.Metadata, idField)
	}
	cell.Metadata[RunmeIdField] = id
}

func CellKindToBlockKind(kind parserv1.CellKind) agent.BlockKind {
	switch kind {
	case parserv1.CellKind_CELL_KIND_CODE:
		return agent.BlockKind_CODE
	case parserv1.CellKind_CELL_KIND_MARKUP:
		return agent.BlockKind_MARKUP
	default:
		return agent.BlockKind_UNKNOWN_BLOCK_KIND
	}
}

func CellOutputToBlockOutput(output *parserv1.CellOutput) (*agent.BlockOutput, error) {
	if output == nil {
		return nil, errors.New("CellOutput is nil")
	}

	boutput := &agent.BlockOutput{
		Items: make([]*agent.BlockOutputItem, 0, len(output.Items)),
	}

	for _, oi := range output.Items {
		boi := &agent.BlockOutputItem{
			Mime:     oi.Mime,
			TextData: string(oi.Data),
		}
		boutput.Items = append(boutput.Items, boi)
	}

	return boutput, nil
}
