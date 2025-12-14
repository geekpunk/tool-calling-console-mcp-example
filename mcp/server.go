package mcp

import (
	"bufio"
	"devtool/config"
	"devtool/logger"
	"devtool/tools"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"io"
	"net"
	"os"
	"sync"
)

// JSON-RPC types
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{}   `json:"id"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCP types
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type CallToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Server struct {
	Config     *config.Config
	ConfigFile string
	mu         sync.RWMutex
}

func NewServer(cfg *config.Config, configFile string) *Server {
	return &Server{
		Config:     cfg,
		ConfigFile: configFile,
	}
}

func (s *Server) WatchConfig() {
	if s.ConfigFile == "" {
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("Failed to create file watcher: %v", err)
		return
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					logger.Info("Config file modified. Reloading...")
					newCfg, err := config.LoadConfig(s.ConfigFile)
					if err != nil {
						logger.Error("Failed to reload config: %v", err)
						continue
					}
					
					s.mu.Lock()
					s.Config = newCfg
					s.mu.Unlock()
					logger.Info("Configuration reloaded successfully.")
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Error("Watcher error: %v", err)
			}
		}
	}()

	err = watcher.Add(s.ConfigFile)
	if err != nil {
		logger.Error("Failed to watch config file: %v", err)
	} else {
		logger.Info("Watching config file: %s", s.ConfigFile)
	}
}

func (s *Server) ServeStdio() {
	s.WatchConfig()
	ip := getLocalIP()
	logger.Info("MCP Server started. Status: Running. Mode: Stdio. IP: %s", ip)
	s.serveStream(os.Stdin, os.Stdout)
}

func (s *Server) ServeTCP(port int) {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("Failed to start TCP server: %v", err)
		os.Exit(1)
	}
	s.WatchConfig()

	ip := getLocalIP()
	// Get actual port if 0 was passed
	port = listener.Addr().(*net.TCPAddr).Port
	logger.Info("MCP Server started. Status: Running. Mode: TCP. IP: %s Port: %d", ip, port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Accept error: %v", err)
			continue
		}
		go s.serveStream(conn, conn)
	}
}

func (s *Server) serveStream(r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	// Increase buffer size just in case
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			// Log error to stderr, don't crash
			logger.Error("Failed to parse JSON: %v", err)
			continue
		}

		s.handleRequest(req, w)
	}
}

func (s *Server) handleRequest(req JSONRPCRequest, w io.Writer) {
	var resp JSONRPCResponse
	resp.JSONRPC = "2.0"
	resp.ID = req.ID

	switch req.Method {
	case "initialize":
		resp.Result = map[string]interface{}{
			"protocolVersion": "2024-11-05", // Example version
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]string{
				"name":    "devtool",
				"version": "0.1.0",
			},
		}
	case "notifications/initialized":
		// No response needed for notifications
		return
	case "tools/list":
		toolList := []Tool{}
		
		s.mu.RLock()
		tools := s.Config.Tools
		workflows := s.Config.Workflows
		s.mu.RUnlock()

		for _, t := range tools {
			props := make(map[string]interface{})
			required := []string{}
			for _, p := range t.Parameters {
				props[p.Name] = map[string]string{
					"type":        p.Type,
					"description": p.Description,
				}
				if p.Required {
					required = append(required, p.Name)
				}
			}

			toolList = append(toolList, Tool{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: InputSchema{
					Type:       "object",
					Properties: props,
					Required:   required,
				},
			})
		}
		

		
		// Add Workflows
		for _, w := range workflows {
			props := make(map[string]interface{})
			required := []string{}
			for _, p := range w.Parameters {
				props[p.Name] = map[string]string{
					"type":        p.Type,
					"description": p.Description,
				}
				if p.Required {
					required = append(required, p.Name)
				}
			}

			toolList = append(toolList, Tool{
				Name:        w.Name,
				Description: w.Description,
				InputSchema: InputSchema{
					Type:       "object",
					Properties: props,
					Required:   required,
				},
			})
		}

		resp.Result = map[string]interface{}{
			"tools": toolList,
		}
	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &JSONRPCError{Code: -32700, Message: "Parse error"}
			break
		}

		// Find tool or workflow
		var selectedTool *config.ToolConfig
		var selectedWorkflow *config.WorkflowConfig

		s.mu.RLock()
		cfgTools := s.Config.Tools
		cfgWorkflows := s.Config.Workflows
		s.mu.RUnlock()

		for _, t := range cfgTools {
			if t.Name == params.Name {
				selectedTool = &t
				break
			}
		}

		if selectedTool == nil {
			for _, w := range cfgWorkflows {
				if w.Name == params.Name {
					selectedWorkflow = &w
					break
				}
			}
		}

		if selectedTool == nil && selectedWorkflow == nil {
			resp.Error = &JSONRPCError{Code: -32601, Message: "Tool or Workflow not found"}
			break
		}

		// Execute
		logger.Info("Executing %s with params: %v", params.Name, params.Arguments)
		
		var output string
		var err error

		if selectedTool != nil {
			output, err = tools.ExecuteTool(*selectedTool, params.Arguments)
		} else {
			// Note: We are passing cfgTools to ExecuteWorkflow, if ExecuteWorkflow does not modify the slice/map
			// it should be fine. However, since we are under RLock above, we copied the slice headers.
			// Ideally ExecuteWorkflow should be safe.
			output, err = tools.ExecuteWorkflow(*selectedWorkflow, cfgTools, params.Arguments)
		}

		isError := false
		if err != nil {
			isError = true
			output = fmt.Sprintf("Error: %v\nOutput: %s", err, output)
			logger.Error("Execution %s finished with error: %v. Output: %s", params.Name, err, output)
		} else {
			logger.Info("Execution %s finished successfully.", params.Name)
		}

		resp.Result = CallToolResult{
			Content: []Content{
				{Type: "text", Text: output},
			},
			IsError: isError,
		}

	default:
		// Ignore unknown notifications, return error for unknown requests with ID
		if req.ID != nil {
			resp.Error = &JSONRPCError{Code: -32601, Message: "Method not found"}
		} else {
			return
		}
	}

	// Send response
	bytes, err := json.Marshal(resp)
	if err != nil {
		logger.Error("Failed to marshal response: %v", err)
		return
	}
	// Write to the specific writer
	w.Write(bytes)
	w.Write([]byte("\n"))
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "unknown"
}
