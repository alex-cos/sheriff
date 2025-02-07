package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the top-level application configuration loaded from a YAML file.
type Config struct {
	StopTimeout time.Duration   `yaml:"stopTimeout"`
	LogLevel    string          `yaml:"logLevel"`
	Monitor     MonitorConfig   `yaml:"monitor"`
	Services    []ServiceConfig `yaml:"services"`
}

// MonitorConfig holds settings for the periodic service monitor.
type MonitorConfig struct {
	Period  time.Duration `yaml:"period"`
	Restart bool          `yaml:"restart"`
}

// ServiceConfig defines a single managed service with its command, arguments,
// dependencies, restart policy, and retry count.
type ServiceConfig struct {
	Name        string   `yaml:"name"`
	Command     string   `yaml:"command"`
	Argument    []string `yaml:"arguments"`
	DependsOn   []string `yaml:"dependsOn"`
	MaxRetries  int      `yaml:"maxRetries"`
	RestartDeps bool     `yaml:"restartDeps"`
}

// Load reads a YAML file, unmarshals it into the Config struct, and runs validation.
func (c *Config) Load(filename string) error {
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read configuration file '%s': %w", filename, err)
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return fmt.Errorf("failed to decode configuration file '%s': %w", filename, err)
	}

	return c.Validate()
}

// Validate checks that the configuration is consistent: at least one service,
// valid log level, non-negative timeouts, unique non-empty service names,
// and every service has a command.
func (c *Config) Validate() error {
	if len(c.Services) == 0 {
		return errors.New("configuration must define at least one service")
	}

	if c.StopTimeout < 0 {
		return errors.New("stopTimeout must be non-negative")
	}

	if c.Monitor.Period < 0 {
		return errors.New("monitor.period must be non-negative")
	}

	switch c.LogLevel {
	case "debug", "info", "warn", "error", "":
	default:
		return fmt.Errorf("invalid logLevel '%s': must be debug, info, warn, or error", c.LogLevel)
	}

	names := make(map[string]bool, len(c.Services))
	for _, s := range c.Services {
		if s.Name == "" {
			return errors.New("each service must have a non-empty name")
		}
		if names[s.Name] {
			return fmt.Errorf("duplicate service name '%s'", s.Name)
		}
		names[s.Name] = true

		if s.Command == "" {
			return fmt.Errorf("service '%s': command is required", s.Name)
		}
	}

	return nil
}
