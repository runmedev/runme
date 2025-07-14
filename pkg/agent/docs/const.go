package docs

const (
	BASHLANG = "bash"
	// OUTPUTLANG is the language to give to output code cells.
	// We want to potentially distinguish output from code cells because output cells are nested inside cells
	// in notebooks. Therefore if we want to be able to convert a markdown document into a document with cells
	// then having a unique language for output cells helps us identify them and properly reencode them.
	OUTPUTLANG = "output"

	// StatefulRunmeOutputItemsMimeType is the mime type for output items in runme. This will be a JSON object.
	// See:
	//    https://github.com/stateful/vscode-runme/blob/3e36b16e3c41ad0fa38f0197f1713135e5edb27b/src/constants.ts#L6
	//    https://github.com/jlewi/foyle/issues/286
	StatefulRunmeOutputItemsMimeType = "stateful.runme/output-items"
	StatefulRunmeTerminalMimeType    = "stateful.runme/terminal"
	VSCodeNotebookStdOutMimeType     = "application/vnd.code.notebook.stdout"
	VSCodeNotebookStdErrMimeType     = "application/vnd.code.notebook.stderr"
)
