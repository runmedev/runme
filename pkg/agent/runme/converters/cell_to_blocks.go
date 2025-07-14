package converters

import (
	"github.com/pkg/errors"

	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
)

// Doc represents a document
// This is a hack for when we copied this code over from foyle; if we keep doc; we should make it a proto.
type Doc struct {
	Cells []*parserv1.Cell
}

// NotebookToDoc converts a runme Notebook to a foyle Doc
func NotebookToDoc(nb *parserv1.Notebook) (*Doc, error) {
	if nb == nil {
		return nil, errors.New("Notebook is nil")
	}

	doc := &Doc{
		Cells: make([]*parserv1.Cell, 0, len(nb.Cells)),
	}

	for _, cell := range nb.Cells {
		cell, err := CellToCell(cell)
		if err != nil {
			return nil, err
		}
		doc.Cells = append(doc.Cells, cell)
	}

	return doc, nil
}

// CellToCell converts a runme Cell to a foyle Cell
//
// N.B. cell metadata is currently ignored.
func CellToCell(cell *parserv1.Cell) (*parserv1.Cell, error) {
	if cell == nil {
		return nil, errors.New("Cell is nil")
	}

	cOutputs := make([]*parserv1.CellOutput, 0, len(cell.Outputs))
	cOutputs = append(cOutputs, cell.Outputs...)
	cellKind := CellKindToCellKind(cell.Kind)

	id := ""
	if cell.Metadata != nil {
		newId := GetCellID(cell)
		if newId != "" {
			id = newId
		}
	}

	return &parserv1.Cell{
		RefId:      id,
		LanguageId: cell.LanguageId,
		Value:      cell.Value,
		Kind:       cellKind,
		Outputs:    cOutputs,
		Metadata:   cell.Metadata,
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

	if cell.RefId != "" {
		return cell.RefId
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

func CellKindToCellKind(kind parserv1.CellKind) parserv1.CellKind {
	switch kind {
	case parserv1.CellKind_CELL_KIND_CODE:
		return parserv1.CellKind_CELL_KIND_CODE
	case parserv1.CellKind_CELL_KIND_MARKUP:
		return parserv1.CellKind_CELL_KIND_MARKUP
	default:
		return parserv1.CellKind_CELL_KIND_UNSPECIFIED
	}
}

func CellOutputToCellOutput(output *parserv1.CellOutput) (*parserv1.CellOutput, error) {
	if output == nil {
		return nil, errors.New("CellOutput is nil")
	}

	coutput := &parserv1.CellOutput{
		Items: make([]*parserv1.CellOutputItem, 0, len(output.Items)),
	}

	for _, oi := range output.Items {
		coi := &parserv1.CellOutputItem{
			Mime: oi.Mime,
			Data: oi.Data,
		}
		coutput.Items = append(coutput.Items, coi)
	}

	return coutput, nil
}
