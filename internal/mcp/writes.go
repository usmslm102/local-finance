package mcp

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Write tools share strict argument validation and the existing result envelope.
func registerWriteTool(server *sdk.Server, name, description string, properties map[string]any, required []string, save func(json.RawMessage) (any, error)) {
	registerMutationTool(server, name, description, properties, required, false, save)
}

func registerMutationTool(server *sdk.Server, name, description string, properties map[string]any, required []string, destructive bool, save func(json.RawMessage) (any, error)) {
	schema := map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
	encoded, _ := json.Marshal(schema)
	var inputSchema jsonschema.Schema
	if err := json.Unmarshal(encoded, &inputSchema); err != nil {
		panic(err)
	}
	resolved, err := inputSchema.Resolve(nil)
	if err != nil {
		panic(err)
	}
	closed := false
	server.AddTool(&sdk.Tool{Name: name, Description: description, InputSchema: schema,
		Annotations: &sdk.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &destructive, IdempotentHint: false, OpenWorldHint: &closed}}, func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		var input any
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return toolError(errors.New("invalid JSON arguments")), nil
		}
		if err := resolved.Validate(input); err != nil {
			return toolError(errors.New("invalid arguments; check required fields, types and supported values")), nil
		}
		if ctx.Err() != nil {
			return toolError(errors.New("request cancelled")), nil
		}
		data, err := save(req.Params.Arguments)
		if err != nil {
			return toolError(err), nil
		}
		// Once saved, report success even if cancellation arrived during the write.
		out := toolOutput{Data: data}
		content, err := json.Marshal(out)
		if err != nil {
			return toolError(errors.New("unable to encode saved data")), nil
		}
		return &sdk.CallToolResult{StructuredContent: out, Content: []sdk.Content{&sdk.TextContent{Text: string(content)}}}, nil
	})
}
