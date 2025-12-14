package main

import (
	"bufio"
	"devtool/config"
	"devtool/logger"
	"devtool/mcp"
	"devtool/tools"
	"encoding/json"
	"flag"
	"fmt"

	"net"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Common flags
	var configPath string

	// Define flags for subcommands
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	serveCmd.StringVar(&configPath, "config", "devtool.yaml", "Path to configuration file")
	servePort := serveCmd.Int("port", 0, "Port to listen on (0 for Stdio, >0 for TCP)")
	serveLog := serveCmd.String("logfile", "", "Path to log file")

	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	runCmd.StringVar(&configPath, "config", "devtool.yaml", "Path to configuration file")
	runLog := runCmd.String("logfile", "", "Path to log file")

	testCmd := flag.NewFlagSet("test", flag.ExitOnError)
	testAddr := testCmd.String("addr", "", "Address of running MCP server (e.g. localhost:3000)")
	testLog := testCmd.String("logfile", "", "Path to log file")
	testWorkflow := testCmd.String("workflow", "", "Name of the workflow/tool to test")

	switch command {
	case "serve":
		serveCmd.Parse(os.Args[2:])
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		setupLogging(*serveLog, cfg)

		server := mcp.NewServer(cfg, configPath)

		port := *servePort
		if port == 0 && cfg.Server.Port > 0 {
			port = cfg.Server.Port
		}

		if port > 0 {
			server.ServeTCP(port)
		} else {
			server.ServeStdio()
		}

	case "run":
		runCmd.Parse(os.Args[2:])
		args := runCmd.Args()
		if len(args) < 1 {
			fmt.Println("Error: Tool name required")
			os.Exit(1)
		}
		toolName := args[0]

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		setupLogging(*runLog, cfg)

		// Find tool
		var selectedTool *config.ToolConfig
		var selectedWorkflow *config.WorkflowConfig

		for _, t := range cfg.Tools {
			if t.Name == toolName {
				selectedTool = &t
				break
			}
		}

		if selectedTool == nil {
			// Check workflows
			for _, w := range cfg.Workflows {
				if w.Name == toolName {
					selectedWorkflow = &w
					break
				}
			}
		}

		if selectedTool == nil && selectedWorkflow == nil {
			logger.Error("Tool or Workflow '%s' not found in config", toolName)
			os.Exit(1)
		}

		// Parse remaining args as key=value
		toolArgs := make(map[string]interface{})
		for _, arg := range args[1:] {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				toolArgs[parts[0]] = parts[1]
			}
		}

		var output string

		if selectedTool != nil {
			output, err = tools.ExecuteTool(*selectedTool, toolArgs)
		} else {
			output, err = tools.ExecuteWorkflow(*selectedWorkflow, cfg.Tools, toolArgs)
		}

		if err != nil {
			logger.Error("Error executing %s: %v\nOutput: %v", toolName, err, output)
			os.Exit(1)
		}
		fmt.Println(output)

	case "test":
		testCmd.Parse(os.Args[2:])

		// Load config just for defaults (logfile, port)
		cfg, err := config.LoadConfig(configPath)
		var cfgPtr *config.Config
		if err == nil {
			cfgPtr = cfg
		}

		setupLogging(*testLog, cfgPtr)

		addr := *testAddr
		if addr == "" {
			// Try to load config to get default port
			if cfg != nil && cfg.Server.Port > 0 {
				addr = fmt.Sprintf("localhost:%d", cfg.Server.Port)
			}
		}

		if addr == "" {
			fmt.Println("Error: --addr required for test (e.g. localhost:3000)")
			os.Exit(1)
		}
		runTest(addr, *testWorkflow)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  devtool serve --config <path> [--port <port>] [--logfile <path>]")
	fmt.Println("  devtool run <tool-name> [key=value ...] --config <path> [--logfile <path>]")
	fmt.Println("  devtool test --addr <host:port> [--logfile <path>] [--workflow <name>]")
}

func setupLogging(logPath string, cfg *config.Config) {
	path := logPath
	if path == "" && cfg != nil && cfg.LogFile != "" {
		path = cfg.LogFile
	}
	if err := logger.Setup(path); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logging: %v\n", err)
	}
}

func runTest(addr string, workflowFilter string) {
	fmt.Printf("Connecting to MCP server at %s...\n", addr)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		logger.Error("Failed to connect to server: %v", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Use connection for both reading and writing
	scanner := bufio.NewScanner(conn)
	// Output writer
	writer := conn

	// Helper to send request
	reqID := 1
	send := func(method string, params interface{}) mcp.JSONRPCResponse {
		req := mcp.JSONRPCRequest{
			JSONRPC: "2.0",
			Method:  method,
			ID:      reqID,
		}
		if params != nil {
			bytes, _ := json.Marshal(params)
			req.Params = bytes
		}
		reqID++

		bytes, _ := json.Marshal(req)
		if _, err := writer.Write(bytes); err != nil {
			logger.Error("Write failed: %v", err)
		}
		writer.Write([]byte("\n"))

		if scanner.Scan() {
			var resp mcp.JSONRPCResponse
			if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
				logger.Error("Parse response failed: %v", err)
			}
			return resp
		}
		return mcp.JSONRPCResponse{}
	}

	// 1. Initialize
	fmt.Println("\n--- Sending initialize ---")
	resp := send("initialize", nil)
	if resp.Error != nil {
		fmt.Printf("Initialize failed: %v\n", resp.Error)
		return
	}
	fmt.Println("Initialize success.")

	// 2. List tools
	fmt.Println("\n--- Listing tools ---")
	resp = send("tools/list", nil)
	if resp.Error != nil {
		fmt.Printf("List tools failed: %v\n", resp.Error)
		return
	}

	var listsResult struct {
		Tools []mcp.Tool `json:"tools"`
	}
	// The result is actually an interface{}, we need to marshal/unmarshal or mapstructure it
	// But since we are in the same codebase, we know the structure but resp.Result is generic.
	rb, _ := json.Marshal(resp.Result)
	json.Unmarshal(rb, &listsResult)

	fmt.Printf("Found %d tools.\n", len(listsResult.Tools))

	// 3. Call each tool
	for _, tool := range listsResult.Tools {
		// Filter if workflow argument is provided
		if workflowFilter != "" && tool.Name != workflowFilter {
			continue
		}

		fmt.Printf("\n--- Testing tool: %s ---\n", tool.Name)

		args := make(map[string]interface{})
		// Provide dummy args for required params
		if tool.InputSchema.Required != nil {
			for _, req := range tool.InputSchema.Required {
				prop := tool.InputSchema.Properties[req].(map[string]interface{})
				pType := prop["type"].(string)

				if pType == "string" {
					args[req] = "test-value"
				} else {
					args[req] = 1
				}
			}
		}

		callParams := mcp.CallToolParams{
			Name:      tool.Name,
			Arguments: args,
		}

		callResp := send("tools/call", callParams)
		if callResp.Error != nil {
			fmt.Printf("Tool call failed: %v\n", callResp.Error)
		} else {
			fmt.Printf("Tool call success. Result: %v\n", callResp.Result)
		}
	}

	fmt.Println("\nTest run completed.")
}
