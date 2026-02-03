# Tools Generation

We want a principled way to generate the JSON schemas use with OpenAI function calling.

The approach we are using is to 

1. Define service in RPCs corresponding to our toolcalls
2. Use the proto plugin https://github.com/redpanda-data/protoc-gen-go-mcp to generate MCP and JSONSchemas for the
   tool definitions

We haven't integrated the running of protoc-gen-go-mcp into bazel yet. So for now we just use buf
to run the generation locally. The tool protos live in api/proto/agent/tools/v1.

To regenerate

1. Download the [protoc-gen-go-mcp plugin](https://github.com/redpanda-data/protoc-gen-go-mcp)
2. Run ./build_tool_mcps.sh

## Important note about OpenAI compatibility

OpenAI JSON schema is not fully compatible with the proto JSON schema
* See this [note](https://github.com/redpanda-data/protoc-gen-go-mcp?tab=readme-ov-file#openai-compatible)
* See [OpenAI docs about supported schemas](https://platform.openai.com/docs/guides/structured-outputs/supported-schemas#supported-schemas)

Therefore we need to convert the JSON produced by OpenAI before we can unmarshal it into our protos.
We have the function

```
func FixOpenAI(descriptor protoreflect.MessageDescriptor, args map[string]any) {
```

To handle this.

# Vendored Mark3labs

I vendored the mcp package in [mark3labs](https://github.com/mark3labs/mcp-go/tree/main/mcp). I did this
because redpanda was using a newer version than what chronosphereio was on and upgrading broke the chronosphereio
MCP server.
