# DevTool - MCP & CLI Tool Runner

DevTool is a **demo project** designed to illustrate how to build a multi-purpose application that functions as both a **Command Line Interface (CLI)** and a **Model Context Protocol (MCP) server**.

This example demonstrates how you can create a single codebase that serves two purposes:
1.  **For Humans**: A CLI tool to automate daily tasks via terminal commands.
2.  **For AI Agents**: An MCP service that exposes the exact same tools to AI assistants (like Claude Desktop), enabling them to perform actions on your behalf.

It allows you to define custom tools and workflows in a simple configuration file.

## Features

-   **Dual Mode**: Run as a CLI tool or an MCP server.
-   **Configurable**: Define tools in a YAML configuration file.
-   **Web Service Integration**: Call any HTTP API (GET, POST, etc.).
-   **Environment Variables**: Support for environment variables in headers (e.g., for API tokens).

## Installation

```bash
go build -o devtool
```

## Configuration

Create a `devtool.yaml` file to define your tools.

```yaml
server:
  port: 3456 # Optional: Default port for TCP server

tools:
  - name: create-pipeline

    description: Creates a new CI/CD pipeline
    url: https://api.ci-service.com/pipelines
    method: POST
    headers:
      Authorization: "Bearer ${CI_TOKEN}"
    parameters:
      - name: name
        type: string
        description: Name of the pipeline
        required: true
      - name: repo_url
        type: string
        description: URL of the repository
        required: true

  - name: get-status
    description: Checks service status
    url: https://api.status.com/current
    method: GET
    parameters: []

  - name: backup-db
    type: shell
    description: Backs up the database
    command: ./scripts/backup.sh --db $DB_NAME
    parameters:
      - name: db_name
        type: string
        description: Database name
        required: true

```

## Usage

### CLI Mode

Run a tool directly from the command line:

```bash
./devtool run create-pipeline name="My Pipeline" repo_url="https://github.com/user/repo"
```

### MCP Server Mode

Start the MCP server to use with AI assistants (like Claude Desktop or other MCP clients):

```bash
./devtool serve
```

The server communicates via Stdio using JSON-RPC 2.0 unless a port is configured.
If a `port` is specified in `devtool.yaml` or via `--port`, it runs over TCP.

When started, it outputs the server status to stderr:
```text
MCP Server started. Status: Running. Mode: TCP. IP: 192.168.1.10 Port: 3456
```

The server watches the configuration file for changes and automatically reloads it.

### Test Mode

Run a built-in integration test. It will connect to the local server defined in `devtool.yaml` (or via `--addr`):

```bash
./devtool test
```
This is useful for verifying your configuration and tool definitions without an external MCP client.

## Project Structure

```text
.
├── config
│   └── config.go       # Configuration loading logic
├── logger
│   └── logger.go       # Logger implementation
├── mcp
│   └── server.go       # MCP server implementation
├── tools
│   ├── executor.go     # Tool execution logic
│   └── workflow.go     # Workflow execution logic
├── devtool.yaml        # Configuration file
├── go.mod
├── go.sum
├── main.go             # Entry point
└── README.md
```
