package tools

import (
	"encoding/json"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// FixOpenAI applies all OpenAI compatibility transformations to convert OpenAI-formatted JSON
// back to standard protobuf-compatible JSON. This includes:
// - Converting map arrays back to objects
// - Converting string representations back to proper JSON for google.protobuf.Value/ListValue/Struct
// see: https://github.com/redpanda-data/protoc-gen-go-mcp/issues/34
func FixOpenAI(descriptor protoreflect.MessageDescriptor, args map[string]any) {
	var rewrite func(msg protoreflect.MessageDescriptor, path []string, obj map[string]any)

	rewrite = func(msg protoreflect.MessageDescriptor, path []string, obj map[string]any) {
		for i := 0; i < msg.Fields().Len(); i++ {
			field := msg.Fields().Get(i)
			name := string(field.Name())
			currentPath := append(path, name)

			if field.IsMap() {
				// Handle map conversion (from array-of-key-value-pairs to object)
				if arr, ok := obj[name].([]any); ok {
					m := make(map[string]any)
					for _, e := range arr {
						if pair, ok := e.(map[string]any); ok {
							k, kOk := pair["key"].(string)
							v, vOk := pair["value"]
							if kOk && vOk {
								m[k] = v
							}
						}
					}
					obj[name] = m
				}
			} else if field.Kind() == protoreflect.MessageKind {
				fullName := string(field.Message().FullName())

				if field.Cardinality() == protoreflect.Repeated {
					if arr, ok := obj[name].([]any); ok {
						for idx, entry := range arr {
							switch fullName {
							case "google.protobuf.Value":
								if str, ok := entry.(string); ok {
									var value any
									if err := json.Unmarshal([]byte(str), &value); err == nil {
										arr[idx] = value
									}
								}
							case "google.protobuf.ListValue":
								if str, ok := entry.(string); ok {
									var listValue []any
									if err := json.Unmarshal([]byte(str), &listValue); err == nil {
										arr[idx] = listValue
									}
								}
							case "google.protobuf.Struct":
								if str, ok := entry.(string); ok {
									var structValue map[string]any
									if err := json.Unmarshal([]byte(str), &structValue); err == nil {
										arr[idx] = structValue
									}
								}
							default:
								if nested, ok := entry.(map[string]any); ok {
									rewrite(field.Message(), currentPath, nested)
								}
							}
						}
						obj[name] = arr
					}
					continue
				}

				// Handle OpenAI string representations of special protobuf types
				switch fullName {
				case "google.protobuf.Value":
					if str, ok := obj[name].(string); ok {
						var value any
						if err := json.Unmarshal([]byte(str), &value); err == nil {
							obj[name] = value
						}
					}
				case "google.protobuf.ListValue":
					if str, ok := obj[name].(string); ok {
						var listValue []any
						if err := json.Unmarshal([]byte(str), &listValue); err == nil {
							obj[name] = listValue
						}
					}
				case "google.protobuf.Struct":
					if str, ok := obj[name].(string); ok {
						var structValue map[string]any
						if err := json.Unmarshal([]byte(str), &structValue); err == nil {
							obj[name] = structValue
						}
					}
				default:
					// Recursively process nested messages
					if nested, ok := obj[name].(map[string]any); ok {
						rewrite(field.Message(), currentPath, nested)
					}
				}
			}
		}
	}

	rewrite(descriptor, nil, args)
}
