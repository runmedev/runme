package chatkit

import (
	"fmt"
	"strings"

	"github.com/openai/openai-go/responses"
)

// maybeConvertAnnotation converts a responses.ResponseOutputTextAnnotationUnion into a Chatkit
// Annotation when we recognize the source. Returns nil when the annotation can't be converted.
func maybeConvertAnnotation(in responses.ResponseOutputTextAnnotationUnion) *Annotation {
	if !strings.HasPrefix(in.FileID, "ks--google_drive") {
		return nil
	}

	parts := strings.Split(in.FileID, "--")
	if len(parts) == 0 {
		return nil
	}

	docID := parts[len(parts)-1]
	if docID == "" {
		return nil
	}

	// TODO(jlewi): How could we make the links open in AISRE?
	// The links point to the ${FILE}.index.json document. It doesn't make sense to lookup each document here.
	// I think what we could do is use an AISRE URL with a special query argument to indicate its a company knowledge
	// annotation. Then in the AISRE app we could look up the Google Drive information; e.g. if its ${FILE}.index.json
	// we could look for the corresponding ${FILE}.json (or potentially add it as metadata or front-matter)
	// and then open that file.
	source := NewUrlSource()
	source.Url = fmt.Sprintf("https://drive.google.com/file/d/%s/view", docID)
	source.Title = in.Filename

	// Attribution is what gets shown as text in inline citations.
	source.Attribution = in.Filename

	annotation := NewAnnotation()
	annotation.Source = source
	if in.JSON.Index.Valid() {
		idx := int(in.Index)
		annotation.Index = &idx
	}

	return &annotation
}
