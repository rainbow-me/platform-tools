package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/rainbow-me/platform-tools/common/env"
	"github.com/rainbow-me/platform-tools/common/logger"
)

const (
	fileFormat     = ".yaml"        // File format of the config files
	relativePath   = "./cmd/config" // Default relative path for config files (base path)
	binaryPath     = "./config"     // Path for binary build config (base path)
	binaryDir      = "target"       // Directory name for the binary target
	binaryInDocker = "app"          // Directory name for Docker deployment
	envVarPrefix   = "env://"       // Prefix for environment variables
)

// YamlReadConfig holds the configuration paths (relative and absolute).
type YamlReadConfig struct {
	RelativePath string // Path relative to the current directory
	AbsolutePath string // Absolute path if provided
	DynamicDir   string // Optional dynamic directory
}

// ReadConfigOption is a function signature used to set configuration options.
type ReadConfigOption func(*YamlReadConfig)

// WithRelativePath sets a relative path for the config file.
func WithRelativePath(path string) ReadConfigOption {
	return func(config *YamlReadConfig) {
		config.RelativePath = path
	}
}

// WithAbsolutePath sets an absolute path for the config file.
func WithAbsolutePath(path string) ReadConfigOption {
	return func(config *YamlReadConfig) {
		config.AbsolutePath = path
	}
}

// WithDynamicDir allows setting a dynamic subdirectory for the configuration path.
func WithDynamicDir(dynamicDir string) ReadConfigOption {
	return func(config *YamlReadConfig) {
		config.DynamicDir = dynamicDir
	}
}

// LoadConfig loads the YAML configuration file based on the environment and provided options.
// It reads from relative or absolute paths and uses Viper to parse the config file.
func LoadConfig(conf interface{}, logger *logger.Logger, options ...ReadConfigOption) error { //nolint:cyclop
	var pathToConfigDir string

	// Default path setup
	config := &YamlReadConfig{RelativePath: relativePath}

	// Apply any configuration options provided
	for _, option := range options {
		option(config)
	}

	// Get the current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		logger.Error("Error getting current working directory", zap.Error(err))

		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	logger.Info("Current working directory", zap.String("directory", currentDir))

	// Adjust config path if running from binary target or Docker container
	if strings.Contains(currentDir, binaryDir) || strings.Contains(currentDir, binaryInDocker) {
		logger.Info("Binary directory", zap.String("directory", binaryDir))
		config.RelativePath = binaryPath
	}

	// Add dynamic directory if provided
	if config.DynamicDir != "" {
		config.RelativePath = fmt.Sprintf("%s/%s", config.RelativePath, config.DynamicDir)
		if config.AbsolutePath != "" {
			config.AbsolutePath = fmt.Sprintf("%s/%s", config.AbsolutePath, config.DynamicDir)
		}
		logger.Info("Updated relative path", zap.String("path", config.RelativePath))
	}

	// Determine whether to use the relative or absolute path for the config file
	if config.AbsolutePath != "" {
		logger.Info("Using absolute path", zap.String("path", config.AbsolutePath))
		pathToConfigDir = config.AbsolutePath
	} else {
		logger.Info("Using relative path", zap.String("path", config.RelativePath))
		pathToConfigDir = config.RelativePath
	}

	// Get the current environment (like 'development', 'production')
	currentEnv, err := env.GetApplicationEnv()
	if err != nil {
		return fmt.Errorf("invalid environment: %w", err)
	}

	// Construct the file path using the environment and the dynamic directory
	filePath := fmt.Sprintf("%s/%s%s", pathToConfigDir, currentEnv, fileFormat)
	logger.Info("Reading config file from path", zap.String("path", filePath))

	// Set up Viper to read the config file
	viper.SetConfigFile(filePath)
	viper.SetEnvPrefix("")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv() // Automatically map environment variables

	// Read the configuration file
	err = viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Replace any environment variable placeholders (e.g., "${ENV_VAR}") with actual values
	for _, key := range viper.AllKeys() {
		value := viper.Get(key)
		setEnvVariableFromString(key, value, logger)
	}

	// Unmarshal the config values into the provided struct
	err = viper.Unmarshal(conf)
	if err != nil {
		return fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return nil
}

func setEnvVariableFromString(key string, value interface{}, logger *logger.Logger) {
	if str, ok := value.(string); ok && strings.HasPrefix(str, envVarPrefix) {
		// Extract the environment variable name (everything after "env://")
		envVar := str[len(envVarPrefix):] // Dynamically extract ENV variable name

		// Get the environment variable value
		envValue, exists := os.LookupEnv(envVar)
		if exists {
			viper.Set(key, envValue)
			logger.Info("set environment variable", zap.String("variableName", envVar))
		} else {
			viper.Set(key, "") // Set to empty string if env var is missing
			logger.Warn("environment variable not found", zap.String("variableName", envVar))
		}
	}
}
