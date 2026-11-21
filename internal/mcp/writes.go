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
	registerMutationToolSchema(server, name, description, schema, destructive, func(_ context.Context, raw json.RawMessage) (any, error) {
		return save(raw)
	})
}

func registerMutationToolSchema(server *sdk.Server, name, description string, schema map[string]any, destructive bool, save func(context.Context, json.RawMessage) (any, error)) {
	registerOperationToolSchema(server, name, description, schema, false, destructive, save)
}

func registerPreviewToolSchema(server *sdk.Server, name, description string, schema map[string]any, read func(context.Context, json.RawMessage) (any, error)) {
	properties := schema["properties"].(map[string]any)
	properties["page"] = map[string]any{"type": "integer", "minimum": 1, "maximum": 1000000}
	properties["page_size"] = map[string]any{"type": "integer", "minimum": 1, "maximum": 200}
	registerOperationToolSchema(server, name, description, schema, true, false, read)
}

func registerOperationToolSchema(server *sdk.Server, name, description string, schema map[string]any, readOnly, destructive bool, save func(context.Context, json.RawMessage) (any, error)) {
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
		Annotations: &sdk.ToolAnnotations{ReadOnlyHint: readOnly, DestructiveHint: &destructive, IdempotentHint: readOnly, OpenWorldHint: &closed}}, func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		var input any
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return toolError(errors.New("invalid JSON arguments")), nil
		}
		if err := resolved.Validate(input); err != nil {
			return toolError(errors.New("invalid arguments; check required fields, types and supported values")), nil
		}
		a := arguments{Page: 1, PageSize: 50}
		if readOnly {
			if err := json.Unmarshal(req.Params.Arguments, &a); err != nil {
				return toolError(errors.New("invalid argument types; pagination must use integer encodings")), nil
			}
		}
		if ctx.Err() != nil {
			return toolError(errors.New("request cancelled")), nil
		}
		data, err := save(ctx, req.Params.Arguments)
		if err != nil {
			return toolError(err), nil
		}
		// Once saved, report success even if cancellation arrived during the write.
		out := toolOutput{Data: data}
		if readOnly {
			out, err = prepareOutput(data, a, false)
			if err != nil {
				return toolError(errors.New("unable to encode preview")), nil
			}
		}
		content, err := json.Marshal(out)
		if err != nil {
			return toolError(errors.New("unable to encode saved data")), nil
		}
		return &sdk.CallToolResult{StructuredContent: out, Content: []sdk.Content{&sdk.TextContent{Text: string(content)}}}, nil
	})
}
