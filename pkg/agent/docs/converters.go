package docs

import (
	"math"
	"strings"

	"github.com/runmedev/runme/v3/internal/ulid"
	"github.com/runmedev/runme/v3/pkg/agent/runme/converters"

	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
	"github.com/runmedev/runme/v3/document/editor"
	"github.com/runmedev/runme/v3/document/identity"
)

const (
	codeTruncationMessage = "<...code was truncated...>"
	truncationMessage     = "<...stdout was truncated...>"
)

// MarkdownToCells converts a markdown string into a sequence of cells.
// This function relies on RunMe's Markdown->Cells conversion; underneath the hood that uses goldmark to walk the AST.
// RunMe's deserialization function doesn't have any notion of output in markdown. However, in Foyle outputs
// are rendered to code cells of language "output". So we need to do some post processing to convert the outputs
// into output items
func MarkdownToCells(mdText string) ([]*parserv1.Cell, error) {
	// N.B. We don't need to add any identities
	resolver := identity.NewResolver(identity.UnspecifiedLifecycleIdentity)
	options := editor.Options{
		IdentityResolver: resolver,
	}
	notebook, err := editor.Deserialize([]byte(mdText), options)

	cells := make([]*parserv1.Cell, 0, len(notebook.Cells))

	var lastCodeCell *parserv1.Cell
	for _, cell := range notebook.Cells {

		var tr *parserv1.TextRange

		if cell.TextRange != nil {
			tr = &parserv1.TextRange{
				Start: uint32(cell.TextRange.Start),
				End:   uint32(cell.TextRange.End),
			}
		}

		id, ok := cell.Metadata[converters.RunmeIdField]
		if !ok {
			id = ulid.GenerateID()
		}

		cellPb := &parserv1.Cell{
			RefId:      id,
			Kind:       parserv1.CellKind(cell.Kind),
			Value:      cell.Value,
			LanguageId: cell.LanguageID,
			Metadata:   cell.Metadata,
			TextRange:  tr,
		}

		c, err := converters.CellToCell(cellPb)
		if err != nil {
			return nil, err
		}

		// We need to handle the case where the cell is an output code cell.
		if cell.Kind == editor.CodeKind {
			if cell.LanguageID == OUTPUTLANG {
				// This is an output cell
				// We need to append the output to the last code cell
				if lastCodeCell != nil {
					if lastCodeCell.Outputs == nil {
						lastCodeCell.Outputs = make([]*parserv1.CellOutput, 0, 1)
					}
					lastCodeCell.Outputs = append(lastCodeCell.Outputs, &parserv1.CellOutput{
						Items: []*parserv1.CellOutputItem{
							{
								Data: []byte(c.Value),
							},
						},
					})
					continue
				}

				// Since we don't have a code cell to add the output to just treat it as a code cell
			} else {
				// Update the lastCodeCell
				lastCodeCell = cellPb
			}
		} else {
			// If we have a non-nil markup cell then we zero out lastCodeCell so that a subsequent output cell
			// wouldn't be added to the last code cell.
			if c.GetValue() != "" {
				lastCodeCell = nil
			}
		}

		cells = append(cells, cellPb)
	}

	return cells, err
}

// CellToMarkdown converts a cell to markdown
// maxLength is a maximum length for the generated markdown. This is a soft limit and may be exceeded slightly
// because we don't account for some characters like the outputLength and the truncation message
// A value <=0 means no limit.
func CellToMarkdown(cell *parserv1.Cell, maxLength int) string {
	sb := strings.Builder{}
	writeCellMarkdown(&sb, cell, maxLength)
	return sb.String()
}

func writeCellMarkdown(sb *strings.Builder, cell *parserv1.Cell, maxLength int) {
	maxInputLength := -1
	maxOutputLength := -1

	if maxLength > 0 {
		// Allocate 50% of the max length for input and output
		// This is crude. Arguably we could be dynamic e.g. if the output is < .5 maxLength we should allocate
		// the unused capacity for inputs. But for simplicity we don't do that. We do allocate unused input capacity
		// to the output. In practice outputs tend to be much longer than inputs. Inputs are human authored
		// whereas outputs are more likely to be produced by a machine (e.g. log output) and therefore very long
		maxInputLength = int(math.Floor(0.5*float64(maxLength)) + 1)
		maxOutputLength = maxInputLength
	}

	switch cell.GetKind() {
	case parserv1.CellKind_CELL_KIND_CODE:
		// Code just gets written as a code cell
		sb.WriteString("```" + BASHLANG + "\n")

		data := cell.GetValue()
		if len(data) > maxInputLength && maxInputLength > 0 {
			data = tailLines(data, maxInputLength)
			data = codeTruncationMessage + "\n" + data

			remaining := maxLength - len(data)
			if remaining > 0 {
				maxOutputLength += remaining
			}
		}
		sb.WriteString(data)
		sb.WriteString("\n```\n")
	default:
		// Otherwise assume its a markdown cell

		data := cell.GetValue()
		if len(data) > maxInputLength && maxInputLength > 0 {
			data = tailLines(data, maxInputLength)
			remaining := maxLength - len(data)
			if remaining > 0 {
				maxOutputLength += remaining
			}
		}
		sb.WriteString(data + "\n")
	}

	// Handle the outputs
	for _, output := range cell.GetOutputs() {
		for _, oi := range output.Items {
			if oi.GetMime() == StatefulRunmeOutputItemsMimeType || oi.GetMime() == StatefulRunmeTerminalMimeType {
				// See: https://github.com/jlewi/foyle/issues/286. This output item contains a JSON dictionary
				// with a bunch of meta information that seems specific to Runme/stateful and not necessarily
				// relevant as context for AI so we filter it out. The output item we are interested in should
				// have a mime type of application/vnd.code.notebook.stdout and contain the stdout of the executed
				// code.
				//
				// We use an exclude list for now because Runme is adding additional mime types as it adds custom
				// renderers. https://github.com/stateful/vscode-runme/blob/3e36b16e3c41ad0fa38f0197f1713135e5edb27b/src/constants.ts#L6
				// So for now we want to error on including useless data rather than silently dropping useful data.
				// In the future we may want to revisit that.
				//
				continue
			}

			sb.WriteString("```" + OUTPUTLANG + "\n")
			textData := string(oi.GetData())
			if 0 < maxOutputLength && len(textData) > maxOutputLength {
				textData = textData[:maxOutputLength]
				sb.WriteString(textData)
				// Don't write a newline before writing truncation because that is more likely to lead to confusion
				// because people might not realize the line was truncated.
				// Emit a message indicating that the output was truncated
				// This is intended for the LLM so it knows that it is working with a truncated output.
				sb.WriteString(truncationMessage)
			} else {
				sb.WriteString(textData)
			}

			sb.WriteString("\n```\n")
		}
	}
}
