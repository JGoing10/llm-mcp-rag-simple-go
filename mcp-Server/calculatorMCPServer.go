package main

import (
	"context"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// 计算器 MCP 服务器：提供基础加减乘除
func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-calculator-server",
		Version: "0.1.0",
	}, nil)

	// 入参定义
	type CalcArgs struct {
		Op string  `json:"op"` // add / sub / mul / div
		A  float64 `json:"a"`
		B  float64 `json:"b"`
	}

	// 注册 calculate 工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "calculate",
		Description: "执行基础加减乘除：op in [add, sub, mul, div]",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"op": map[string]any{
					"type":        "string",
					"description": "操作类型：add/sub/mul/div",
				},
				"a": map[string]any{
					"type":        "number",
					"description": "左操作数",
				},
				"b": map[string]any{
					"type":        "number",
					"description": "右操作数",
				},
			},
			"required": []string{"op", "a", "b"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args CalcArgs) (*mcp.CallToolResult, any, error) {
		var result float64
		switch args.Op {
		case "add":
			result = args.A + args.B
		case "sub":
			result = args.A - args.B
		case "mul":
			result = args.A * args.B
		case "div":
			if args.B == 0 {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: "division by zero"}},
				}, nil, nil
			}
			result = args.A / args.B
		default:
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("unsupported op: %s", args.Op)}},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("%g", result)},
			},
		}, nil, nil
	})

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
