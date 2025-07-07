package docs

import (
	"math"
	"strings"

	"github.com/runmedev/runme/v3/pkg/agent/runme/converters"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
	"github.com/runmedev/runme/v3/document/editor"
	"github.com/runmedev/runme/v3/document/identity"
)

const (
	codeTruncationMessage = "<...code was truncated...>"
	truncationMessage     = "<...stdout was truncated...>"
)

// MarkdownToBlocks converts a markdown string into a sequence of blocks.
// This function relies on RunMe's Markdown->Cells conversion; underneath the hood that uses goldmark to walk the AST.
// RunMe's deserialization function doesn't have any notion of output in markdown. However, in Foyle outputs
// are rendered to code blocks of language "output". So we need to do some post processing to convert the outputs
// into output items
func MarkdownToBlocks(mdText string) ([]*agentv1.Block, error) {
	// N.B. We don't need to add any identities
	resolver := identity.NewResolver(identity.UnspecifiedLifecycleIdentity)
	options := editor.Options{
		IdentityResolver: resolver,
	}
	notebook, err := editor.Deserialize([]byte(mdText), options)

	blocks := make([]*agentv1.Block, 0, len(notebook.Cells))

	var lastCodeBlock *agentv1.Block
	for _, cell := range notebook.Cells {

		var tr *parserv1.TextRange

		if cell.TextRange != nil {
			tr = &parserv1.TextRange{
				Start: uint32(cell.TextRange.Start),
				End:   uint32(cell.TextRange.End),
			}
		}

		cellPb := &parserv1.Cell{
			Kind:       parserv1.CellKind(cell.Kind),
			Value:      cell.Value,
			LanguageId: cell.LanguageID,
			Metadata:   cell.Metadata,
			TextRange:  tr,
		}

		block, err := converters.CellToBlock(cellPb)
		if err != nil {
			return nil, err
		}

		// We need to handle the case where the block is an output code block.
		if block.Kind == agentv1.BlockKind_BLOCK_KIND_CODE {
			if block.Language == OUTPUTLANG {
				// This is an output block
				// We need to append the output to the last code block
				if lastCodeBlock != nil {
					if lastCodeBlock.Outputs == nil {
						lastCodeBlock.Outputs = make([]*agentv1.BlockOutput, 0, 1)
					}
					lastCodeBlock.Outputs = append(lastCodeBlock.Outputs, &agentv1.BlockOutput{
						Items: []*agentv1.BlockOutputItem{
							{
								TextData: block.Contents,
							},
						},
					})
					continue
				}

				// Since we don't have a code block to add the output to just treat it as a code block
			} else {
				// Update the lastCodeBlock
				lastCodeBlock = block
			}
		} else {
			// If we have a non-nil markup block then we zero out lastCodeBlock so that a subsequent output block
			// wouldn't be added to the last code block.
			if block.GetContents() != "" {
				lastCodeBlock = nil
			}
		}

		blocks = append(blocks, block)
	}

	return blocks, err
}

// BlockToMarkdown converts a block to markdown
// maxLength is a maximum length for the generated markdown. This is a soft limit and may be exceeded slightly
// because we don't account for some characters like the outputLength and the truncation message
// A value <=0 means no limit.
func BlockToMarkdown(block *agentv1.Block, maxLength int) string {
	sb := strings.Builder{}
	writeBlockMarkdown(&sb, block, maxLength)
	return sb.String()
}

func writeBlockMarkdown(sb *strings.Builder, block *agentv1.Block, maxLength int) {
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

	switch block.GetKind() {
	case agentv1.BlockKind_BLOCK_KIND_CODE:
		// Code just gets written as a code block
		sb.WriteString("```" + BASHLANG + "\n")

		data := block.GetContents()
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
		// Otherwise assume its a markdown block

		data := block.GetContents()
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
	for _, output := range block.GetOutputs() {
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
			textData := oi.GetTextData()
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
