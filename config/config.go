package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Parameter struct {
	Name        string `yaml:"name" json:"name"`
	Type        string `yaml:"type" json:"type"`
	Description string `yaml:"description" json:"description"`
	Required    bool   `yaml:"required" json:"required"`
}

type ToolConfig struct {
	Name        string `yaml:"name" json:"name"`
	Type        string `yaml:"type" json:"type"` // "http" (default) or "shell"
	Description string `yaml:"description" json:"description"`

	// HTTP specific
	URL     string            `yaml:"url" json:"url"`
	Method  string            `yaml:"method" json:"method"`
	Headers map[string]string `yaml:"headers" json:"headers"`

	// Shell specific
	Command string `yaml:"command" json:"command"`

	Parameters []Parameter `yaml:"parameters" json:"parameters"`
}

type StepConfig struct {
	Name string                 `yaml:"name" json:"name"`
	Tool string                 `yaml:"tool" json:"tool"` // Name of the tool to run
	Args map[string]interface{} `yaml:"args" json:"args"` // Arguments to pass, supports templating
}

type WorkflowConfig struct {
	Name        string       `yaml:"name" json:"name"`
	Description string       `yaml:"description" json:"description"`
	Parameters  []Parameter  `yaml:"parameters" json:"parameters"`
	Steps       []StepConfig `yaml:"steps" json:"steps"`
	Output      string       `yaml:"output" json:"output"` // Output template
}

type ServerConfig struct {
	Port int `yaml:"port" json:"port"`
}

type Config struct {
	LogFile string       `yaml:"logfile" json:"logfile"`
	Server  ServerConfig `yaml:"server" json:"server"`
	Tools     []ToolConfig     `yaml:"tools" json:"tools"`
	Workflows []WorkflowConfig `yaml:"workflows" json:"workflows"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
