package editor

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/runmedev/runme/v3/document"
)

var (
	testDataNested = []byte(`# Examples

It can have an annotation with a name:

` + "```" + `sh {"first":"","name":"echo","second":"2"}
$ echo "Hello, runme!"
` + "```" + `

> bq 1
> bq 2
>
>     echo 1
>
> b1 3

1. Item 1

   ` + "```" + `sh {"first":"","name":"echo-2","second":"2"}
   $ echo "Hello, runme!"
   ` + "```" + `

   First inner paragraph

   Second inner paragraph

2. Item 2
3. Item 3
`)

	testDataNestedFlattened = []byte(`# Examples

It can have an annotation with a name:

` + "```" + `sh {"first":"","name":"echo","second":"2"}
$ echo "Hello, runme!"
` + "```" + `

bq 1
bq 2

    echo 1

b1 3

1. Item 1

` + "```" + `sh {"first":"","name":"echo-2","second":"2"}
$ echo "Hello, runme!"
` + "```" + `

First inner paragraph

Second inner paragraph

2. Item 2

3. Item 3
`)
)

func Test_toCells_DataNested(t *testing.T) {
	doc := document.New(testDataNested, identityResolverAll)
	node, err := doc.Root()
	require.NoError(t, err)
	cells := toCells(doc, node, testDataNested)
	assert.Len(t, cells, 12)
	assert.Equal(t, "# Examples", cells[0].Value)
	assert.Equal(t, "It can have an annotation with a name:", cells[1].Value)
	assert.Equal(t, "$ echo \"Hello, runme!\"", cells[2].Value)
	assert.Equal(t, "bq 1\nbq 2", cells[3].Value)
	assert.Equal(t, "echo 1", cells[4].Value)
	assert.Equal(t, "b1 3", cells[5].Value)
	assert.Equal(t, "1. Item 1", cells[6].Value)
	assert.Equal(t, "$ echo \"Hello, runme!\"", cells[7].Value)
	assert.Equal(t, "First inner paragraph", cells[8].Value)
	assert.Equal(t, "Second inner paragraph", cells[9].Value)
	assert.Equal(t, "2. Item 2", cells[10].Value)
	assert.Equal(t, "3. Item 3", cells[11].Value)
}

func Test_toCells_Lists(t *testing.T) {
	t.Run("ListWithoutCode", func(t *testing.T) {
		data := []byte(`1. Item 1
2. Item 2
3. Item 3
`)
		doc := document.New(data, identityResolverAll)
		node, err := doc.Root()
		require.NoError(t, err)
		cells := toCells(doc, node, data)
		assert.Len(t, cells, 1)
		assert.Equal(t, "1. Item 1\n2. Item 2\n3. Item 3", cells[0].Value)
	})

	t.Run("ListWithCode", func(t *testing.T) {
		data := []byte(`1. Item 1
2. Item 2
   ` + "```sh" + `
   echo 1
   ` + "```" + `
3. Item 3
`)
		doc := document.New(data, identityResolverAll)
		node, err := doc.Root()
		require.NoError(t, err)
		cells := toCells(doc, node, data)
		assert.Len(t, cells, 4)
		assert.Equal(t, "1. Item 1", cells[0].Value)
		assert.Equal(t, "2. Item 2", cells[1].Value)
		assert.Equal(t, "echo 1", cells[2].Value)
		assert.Equal(t, "3. Item 3", cells[3].Value)
	})
}

func Test_toCells_EmptyLang(t *testing.T) {
	data := []byte("```" + `
echo 1
` + "```" + `
`)
	doc := document.New(data, identityResolverAll)
	node, err := doc.Root()
	require.NoError(t, err)
	cells := toCells(doc, node, data)
	assert.Len(t, cells, 1)
	cell := cells[0]
	assert.Equal(t, CodeKind, cell.Kind)
	assert.Equal(t, CellRoleUnspecified, cell.Role)
	assert.Equal(t, "echo 1", cell.Value)
}

func Test_toCells_UnsupportedLang(t *testing.T) {
	data := []byte("```py {\"readonly\":\"true\"}" + `
def hello():
    print("Hello World")
` + "```" + `
`)
	doc := document.New(data, identityResolverAll)
	node, err := doc.Root()
	require.NoError(t, err)
	cells := toCells(doc, node, data)
	assert.Len(t, cells, 1)
	cell := cells[0]
	assert.Equal(t, CodeKind, cell.Kind)
	assert.Equal(t, CellRoleUnspecified, cell.Role)
	assert.Equal(t, "py", cell.LanguageID)
	assert.Equal(t, "true", cell.Metadata["readonly"])
	assert.Equal(t, "def hello():\n    print(\"Hello World\")", cell.Value)
}

func mustReturnSerializedCells(t *testing.T, cells []*Cell, profile string, labelComment bool) []byte {
	t.Helper()

	data, err := serializeCells(cells, profile, labelComment)
	require.NoError(t, err)

	return data
}

func Test_serializeCells_Edited(t *testing.T) {
	data := []byte(`# Examples

1. Item 1
2. Item 2
3. Item 3

Last paragraph.
`)

	parse := func() []*Cell {
		doc := document.New(data, identityResolverAll)
		node, err := doc.Root()
		require.NoError(t, err)
		cells := toCells(doc, node, data)
		assert.Len(t, cells, 3)
		return cells
	}

	t.Run("ChangeInplace", func(t *testing.T) {
		cells := parse()
		cells[0].Value = "# New header"
		assert.Equal(
			t,
			"# New header\n\n1. Item 1\n2. Item 2\n3. Item 3\n\nLast paragraph.\n",
			string(mustReturnSerializedCells(t, cells, OutputProfileDefault, false)),
		)
	})

	t.Run("InsertListItem", func(t *testing.T) {
		cells := parse()
		cells[1].Value = "1. Item 1\n2. Item 2\n3. Item 3\n4. Item 4\n"
		assert.Equal(
			t,
			"# Examples\n\n1. Item 1\n2. Item 2\n3. Item 3\n4. Item 4\n\nLast paragraph.\n",
			string(mustReturnSerializedCells(t, cells, OutputProfileDefault, false)),
		)
	})

	t.Run("AddNewCell", func(t *testing.T) {
		t.Run("First", func(t *testing.T) {
			cells := parse()
			cells = append([]*Cell{
				{
					Kind:     MarkupKind,
					Role:     CellRoleUser,
					Value:    "# Title",
					Metadata: map[string]string{},
				},
			}, cells...)
			assert.Equal(
				t,
				"# Title\n\n# Examples\n\n1. Item 1\n2. Item 2\n3. Item 3\n\nLast paragraph.\n",
				string(mustReturnSerializedCells(t, cells, OutputProfileDefault, false)),
			)
		})

		t.Run("Middle", func(t *testing.T) {
			cells := parse()
			cells = append(cells[:2], cells[1:]...)
			cells[1] = &Cell{
				Kind:     MarkupKind,
				Role:     CellRoleUser,
				Value:    "A new paragraph.\n",
				Metadata: map[string]string{},
			}
			assert.Equal(
				t,
				"# Examples\n\nA new paragraph.\n\n1. Item 1\n2. Item 2\n3. Item 3\n\nLast paragraph.\n",
				string(mustReturnSerializedCells(t, cells, OutputProfileDefault, false)),
			)
		})

		t.Run("Last", func(t *testing.T) {
			cells := parse()
			cells = append(cells, &Cell{
				Kind:     MarkupKind,
				Role:     CellRoleUser,
				Value:    "Paragraph after the last one.",
				Metadata: map[string]string{},
			})
			assert.Equal(
				t,
				"# Examples\n\n1. Item 1\n2. Item 2\n3. Item 3\n\nLast paragraph.\n\nParagraph after the last one.\n",
				string(mustReturnSerializedCells(t, cells, OutputProfileDefault, false)),
			)
		})
	})

	t.Run("RemoveCell", func(t *testing.T) {
		cells := parse()
		cells = append(cells[:1], cells[2:]...)
		assert.Equal(
			t,
			"# Examples\n\nLast paragraph.\n",
			string(mustReturnSerializedCells(t, cells, OutputProfileDefault, false)),
		)
	})
}

func Test_serializeCells_nestedCode(t *testing.T) {
	data := []byte(`# Development

1. Ensure you have [dev](https://github.com/stateful/dev) setup and running

2. Install MacOS dependencies

   ` + "```" + `sh
   brew bundle --no-lock
   ` + "```" + `

3. Setup pre-commit

   ` + "```" + `sh
   pre-commit install
   ` + "```" + `
`)
	doc := document.New(data, identityResolverAll)
	node, err := doc.Root()
	require.NoError(t, err)
	cells := toCells(doc, node, data)
	assert.Equal(
		t,
		`# Development

1. Ensure you have [dev](https://github.com/stateful/dev) setup and running

2. Install MacOS dependencies

`+"```"+`sh
brew bundle --no-lock
`+"```"+`

3. Setup pre-commit

`+"```"+`sh
pre-commit install
`+"```"+`
`,
		string(mustReturnSerializedCells(t, cells, OutputProfileDefault, false)),
	)
}

func Test_serializeCells(t *testing.T) {
	t.Run("attributes", func(t *testing.T) {
		data := []byte("```sh {\"first\":\"\",\"name\":\"echo\",\"second\":\"2\"}\necho 1\n```\n")
		doc := document.New(data, identityResolverAll)
		node, err := doc.Root()
		require.NoError(t, err)
		cells := toCells(doc, node, data)
		assert.Equal(t, string(data), string(mustReturnSerializedCells(t, cells, OutputProfileDefault, false)))
	})

	t.Run("privateFields", func(t *testing.T) {
		data := []byte("```sh {\"first\":\"\",\"name\":\"echo\",\"second\":\"2\"}\necho 1\n```\n")
		doc := document.New(data, identityResolverAll)
		node, err := doc.Root()
		require.NoError(t, err)

		cells := toCells(doc, node, data)
		// Add private fields which will be filtered out during serialization.
		cells[0].Metadata["_private"] = "private"
		cells[0].Metadata["runme.dev/internal"] = "internal"

		assert.Equal(t, string(data), string(mustReturnSerializedCells(t, cells, OutputProfileDefault, false)))
	})

	t.Run("UnsupportedLang", func(t *testing.T) {
		data := []byte(`## Non-Supported Languages

` + "```py {\"readonly\":\"true\"}" + `
def hello():
	print("Hello World")
` + "```" + `
`)
		doc := document.New(data, identityResolverAll)
		node, err := doc.Root()
		require.NoError(t, err)
		cells := toCells(doc, node, data)
		assert.Equal(t, string(data), string(mustReturnSerializedCells(t, cells, OutputProfileDefault, false)))
	})
}

func Test_serializeFencedCodeAttributes(t *testing.T) {
	t.Run("NoMetadata", func(t *testing.T) {
		var buf bytes.Buffer
		serializeFencedCodeAttributes(&buf, &Cell{
			Metadata: nil,
		})
		assert.Equal(t, "", buf.String())
	})

	t.Run("OnlyPrivateMetadata", func(t *testing.T) {
		var buf bytes.Buffer
		serializeFencedCodeAttributes(&buf, &Cell{
			Metadata: map[string]string{
				"_key":              "_value",
				"runme.dev/private": "private",
				"index":             "index",
			},
		})
		assert.Equal(t, "", buf.String())
	})

	t.Run("NamePriority", func(t *testing.T) {
		var buf bytes.Buffer
		serializeFencedCodeAttributes(&buf, &Cell{
			Metadata: map[string]string{
				"a":    "a",
				"b":    "b",
				"c":    "c",
				"name": "name",
			},
		})
		assert.Equal(t, ` {"a":"a","b":"b","c":"c","name":"name"}`, buf.String())
	})
}

// todo(sebastian): Use JSON until we support deserialization
var (
	testDataOutputsText      = []byte(`{"kind":2,"role":1,"value":"$ printf \"\\u001b[34mDoes it work?\\n\"\n$ sleep 2\n$ printf \"\\u001b[32mYes, success!\\x1b[0m\\n\"\n$ exit 16","languageId":"sh","metadata":{"background":"false","id":"01HF7B0KJPF469EG9ZVX256S75","interactive":"true"},"outputs":[{"items":[{"value":"\u001b[34mDoes it work?\r\n\u001b[32mYes, success!\u001b[1B\u001b[13D\u001b[0m","data":"","type":"Buffer","mime":"application/vnd.code.notebook.stdout"}],"processInfo":{"exitReason":{"type":"exit","code":16},"pid":0}}],"executionSummary":{"executionSummary":0,"success":false,"timing":{"startTime":1701280699458,"endTime":1701280701754}}}`)
	testDataOutputsTextEmpty = []byte(`{"kind":2,"role":1,"value":"$ printf \"\\u001b[34mDoes it work?\\n\"\n$ sleep 2\n$ printf \"\\u001b[32mYes, success!\\x1b[0m\\n\"\n$ exit 16","languageId":"sh","metadata":{"background":"false","id":"01HF7B0KJPF469EG9ZVX256S75","interactive":"true"},"outputs":[{"items":[{"value":"","data":"","type":"Buffer","mime":"application/vnd.code.notebook.stdout"}],"processInfo":{"exitReason":{"type":"exit","code":16},"pid":0}}],"executionSummary":{"executionSummary":0,"success":false,"timing":{"startTime":1701280699458,"endTime":1701280701754}}}`)
	testDataOutputsImage     = []byte(`{"kind":2,"role":1,"value":"$ curl -s \"https://runme.dev/runme_cube.svg\"","languageId":"sh","metadata":{"id":"01HF7B0KJPF469EG9ZW030N7RQ","interactive":"false","mimeType":"image/svg+xml"},"outputs":[{"items":[{"value":"","data":"PHN2ZyB3aWR0aD0iOTUzNSIgaGVpZ2h0PSI4MDA0IiB2aWV3Qm94PSIwIDAgOTUzNSA4MDA0IiBmaWxsPSJub25lIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPgo8cGF0aCBkPSJNNTA0Ni45NCAzMy43Mzk1QzQ4MzIuNDYgLTE2LjQwNjEgNDQ5MS42OSAtOS43ODk5IDQyODYuNTkgNDcuODI2NkwzNTA0IDI2OC42MUwyNzIxLjQyIDQ4OS4zOTJMMTE1Ni4yNSA5MzAuOTU4Qzk1MS4zMSA5ODguNTc1IDk1OC4wOTIgMTA3Ni40OSAxMTcyLjQ2IDExMjYuNDdMNDQzMS44NCAxODg3Ljk0QzQ2NDUuNDcgMTkzNy43OSA0OTg2LjE1IDE5MzEuMzQgNTE5MS4wOSAxODczLjU1TDU5NzMuNzUgMTY1Mi43Nkw2NzU2LjQyIDE0MzEuOTdMNzUzOS4wOSAxMjExLjE4TDgzMjEuNzYgOTkwLjM5NEM4NTI2LjcgOTMyLjc3OCA4NTE5LjkyIDg0NS4wNTcgODMwNi4zOCA3OTUuMjE1TDUwNDYuOTQgMzMuNzM5NVoiIGZpbGw9IiM1QjNBREYiIGZpbGwtb3BhY2l0eT0iMC43Ii8","type":"Buffer","mime":"image/svg+xml"}],"processInfo":null}],"executionSummary":{"executionSummary":0,"success":true,"timing":{"startTime":1701282636792,"endTime":1701282636923}}}`)
)

func Test_serializeOutputs(t *testing.T) {
	t.Run("Text", func(t *testing.T) {
		var testCell Cell
		json.Unmarshal(testDataOutputsText, &testCell)

		var buf bytes.Buffer
		serializeCellOutputsText(&buf, &testCell)
		assert.Equal(t, "\n# Ran on 2023-11-29 17:58:19Z for 2.296s exited with 16\nDoes it work?\r\nYes, success!\n", buf.String())
	})

	t.Run("Empty text", func(t *testing.T) {
		var testCell Cell
		json.Unmarshal(testDataOutputsTextEmpty, &testCell)

		var buf bytes.Buffer
		serializeCellOutputsText(&buf, &testCell)
		assert.Equal(t, "\n# Ran on 2023-11-29 17:58:19Z for 2.296s exited with 16\n", buf.String())
	})

	t.Run("Image", func(t *testing.T) {
		var testCell Cell
		json.Unmarshal(testDataOutputsImage, &testCell)

		var buf bytes.Buffer
		serializeCellOutputsImage(&buf, &testCell)
		assert.Equal(t, "\n\n![$ curl -s \"https://runme.dev/runme_cube.svg\"](data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iOTUzNSIgaGVpZ2h0PSI4MDA0IiB2aWV3Qm94PSIwIDAgOTUzNSA4MDA0IiBmaWxsPSJub25lIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPgo8cGF0aCBkPSJNNTA0Ni45NCAzMy43Mzk1QzQ4MzIuNDYgLTE2LjQwNjEgNDQ5MS42OSAtOS43ODk5IDQyODYuNTkgNDcuODI2NkwzNTA0IDI2OC42MUwyNzIxLjQyIDQ4OS4zOTJMMTE1Ni4yNSA5MzAuOTU4Qzk1MS4zMSA5ODguNTc1IDk1OC4wOTIgMTA3Ni40OSAxMTcyLjQ2IDExMjYuNDdMNDQzMS44NCAxODg3Ljk0QzQ2NDUuNDcgMTkzNy43OSA0OTg2LjE1IDE5MzEuMzQgNTE5MS4wOSAxODczLjU1TDU5NzMuNzUgMTY1Mi43Nkw2NzU2LjQyIDE0MzEuOTdMNzUzOS4wOSAxMjExLjE4TDgzMjEuNzYgOTkwLjM5NEM4NTI2LjcgOTMyLjc3OCA4NTE5LjkyIDg0NS4wNTcgODMwNi4zOCA3OTUuMjE1TDUwNDYuOTQgMzMuNzM5NVoiIGZpbGw9IiM1QjNBREYiIGZpbGwtb3BhY2l0eT0iMC43Ii8)\n", buf.String())
	})
}

func Test_removeLabelCommentOnCells(t *testing.T) {
	data := []byte("```sh {\"first\":\"\",\"name\":\"label-in-comments\",\"second\":\"2\"}\n### Exported in runme.dev as label-in-comments\necho 1\n```\n")
	doc := document.New(data, identityResolverNone)
	node, err := doc.Root()
	require.NoError(t, err)

	cells := toCells(doc, node, data)

	require.NotContains(t, cells[0].Value, "### Exported in runme.dev as label-in-comments")
}

func Test_serializeCells_AgentProfile(t *testing.T) {
	cells := []*Cell{
		{
			Kind:  MarkupKind,
			Role:  CellRoleUser,
			Value: "what's running in us-west-1 ec2?",
			Metadata: map[string]string{
				"refId": "user_d5cc7eb1-ef80-4fe0-b2a6-6f8115667508",
			},
		},
		{
			Kind:       CodeKind,
			Role:       CellRoleAssistant,
			Value:      "aws ec2 describe-instances --region us-west-1 --query 'Reservations[].Instances[].{InstanceId:InstanceId,State:State.Name,Tags:Tags,InstanceType:InstanceType,PublicIpAddress:PublicIpAddress,PrivateIpAddress:PrivateIpAddress}' --output json",
			LanguageID: "shell",
			Metadata: map[string]string{
				"runme.dev/id": "fc_0e6c55f00f30e6ae00691e30cb7cdc81a285972219dac4cb34",
			},
			Outputs: []*CellOutput{},
		},
		{
			Kind:  MarkupKind,
			Role:  CellRoleUser,
			Value: "ditch tags",
			Metadata: map[string]string{
				"refId": "user_7aaaef20-0a0c-465b-ba27-2d19d02404df",
			},
		},
		{
			Kind:       CodeKind,
			Role:       CellRoleAssistant,
			Value:      "aws ec2 describe-instances --region us-west-1 --query 'Reservations[].Instances[].{InstanceId:InstanceId,State:State.Name,InstanceType:InstanceType,PublicIpAddress:PublicIpAddress,PrivateIpAddress:PrivateIpAddress}' --output json",
			LanguageID: "shell",
			Metadata: map[string]string{
				"runme.dev/id": "fc_0e6c55f00f30e6ae00691e30d049f481a2b423e4ea43774022",
			},
			Outputs: []*CellOutput{
				{
					Items: []*CellOutputItem{
						{
							Data: "",
							Type: "Buffer",
							Mime: "application/vnd.code.notebook.stdout",
						},
					},
					ProcessInfo: &CellOutputProcessInfo{
						ExitReason: &ProcessInfoExitReason{
							Type: "exit",
							Code: 0,
						},
						Pid: 54333,
					},
				},
			},
			ExecutionSummary: &CellExecutionSummary{
				ExecutionOrder: 1,
				Success:        true,
				Timing: &ExecutionSummaryTiming{
					StartTime: 1763586257383,
					EndTime:   1763586262148,
				},
			},
		},
		{
			Kind:  MarkupKind,
			Role:  CellRoleAssistant,
			Value: "The following EC2 instance is currently running in the us-west-1 region:\n\n- InstanceId: i-046bd1982e81573e8\n- State: running\n- InstanceType: t3.micro\n- Public IP Address: 3.101.34.68\n- Private IP Address: 172.31.24.191\n\nLet me know if you need more details or actions performed on this instance!",
			Metadata: map[string]string{
				"runme.dev/id": "msg_0e6c55f00f30e6ae00691e30d591a881a2bf88ba29a91cd4c3",
			},
		},
	}

	t.Run("SerializeAgentProfile", func(t *testing.T) {
		expected := "> what's running in us-west-1 ec2?\n\n```shell\naws ec2 describe-instances --region us-west-1 --query 'Reservations[].Instances[].{InstanceId:InstanceId,State:State.Name,Tags:Tags,InstanceType:InstanceType,PublicIpAddress:PublicIpAddress,PrivateIpAddress:PrivateIpAddress}' --output json\n```\n\n> ditch tags\n\n```shell\naws ec2 describe-instances --region us-west-1 --query 'Reservations[].Instances[].{InstanceId:InstanceId,State:State.Name,InstanceType:InstanceType,PublicIpAddress:PublicIpAddress,PrivateIpAddress:PrivateIpAddress}' --output json\n\n# Ran on 2025-11-19 21:04:17Z for 4.765s exited with 0\n```\n\nThe following EC2 instance is currently running in the us-west-1 region:\n\n- InstanceId: i-046bd1982e81573e8\n- State: running\n- InstanceType: t3.micro\n- Public IP Address: 3.101.34.68\n- Private IP Address: 172.31.24.191\n\nLet me know if you need more details or actions performed on this instance!\n"
		assert.Equal(t, expected, string(mustReturnSerializedCells(t, cells, OutputProfileAgent, false)))
	})

	t.Run("SerializeDefaultProfile", func(t *testing.T) {
		expected := "what's running in us-west-1 ec2?\n\n```shell\naws ec2 describe-instances --region us-west-1 --query 'Reservations[].Instances[].{InstanceId:InstanceId,State:State.Name,Tags:Tags,InstanceType:InstanceType,PublicIpAddress:PublicIpAddress,PrivateIpAddress:PrivateIpAddress}' --output json\n```\n\nditch tags\n\n```shell\naws ec2 describe-instances --region us-west-1 --query 'Reservations[].Instances[].{InstanceId:InstanceId,State:State.Name,InstanceType:InstanceType,PublicIpAddress:PublicIpAddress,PrivateIpAddress:PrivateIpAddress}' --output json\n\n# Ran on 2025-11-19 21:04:17Z for 4.765s exited with 0\n```\n\nThe following EC2 instance is currently running in the us-west-1 region:\n\n- InstanceId: i-046bd1982e81573e8\n- State: running\n- InstanceType: t3.micro\n- Public IP Address: 3.101.34.68\n- Private IP Address: 172.31.24.191\n\nLet me know if you need more details or actions performed on this instance!\n"
		assert.Equal(t, expected, string(mustReturnSerializedCells(t, cells, OutputProfileDefault, false)))
	})
}
