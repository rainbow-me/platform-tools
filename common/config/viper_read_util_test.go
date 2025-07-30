package config

import (
	"github.com/rainbow-me/platform-tools/common/env"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"os"
	"path/filepath"
	"testing"
)

type TestConfig struct {
	Temporal struct {
		HostPort string `mapstructure:"hostPort"`
	} `mapstructure:"temporal"`
}

// createTempConfig creates a temporary config directory and file, registering cleanup.
func createTempConfig(t *testing.T, basePath, dynamicDir, appEnv, content string) string {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "config-test-*")
	require.NoError(t, err)

	configPath := filepath.Join(tempDir, basePath)
	if dynamicDir != "" {
		configPath = filepath.Join(configPath, dynamicDir)
	}
	err = os.MkdirAll(configPath, 0755)
	require.NoError(t, err)

	if content != "" {
		filePath := filepath.Join(configPath, appEnv+".yaml")
		err = os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })
	return tempDir
}

// TestBasicConfigurationLoading tests loading a basic YAML configuration.
func TestBasicConfigurationLoading(t *testing.T) {
	appEnv := "development"
	yamlContent := `
temporal:
hostPort: "localhost:7233"
`
	basePath := "cmd/config"
	dynamicDir := ""
	chdirTo := ""
	options := []ReadConfigOption(nil)
	expectedHostPort := "localhost:7233"

	tempDir := createTempConfig(t, basePath, dynamicDir, appEnv, yamlContent)

	err := os.Setenv(env.ApplicationEnvKey, appEnv)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir
	if chdirTo != "" {
		chdirPath = filepath.Join(tempDir, chdirTo)
		err = os.MkdirAll(chdirPath, 0755)
		require.NoError(t, err)
	}
	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := zaptest.NewLogger(t)

	var conf TestConfig
	err = LoadConfig(&conf, logger, options...)
	require.NoError(t, err)
	require.Equal(t, expectedHostPort, conf.Temporal.HostPort)
}

// TestOverrideWithEnvVar tests overriding a YAML value with an environment variable.
func TestOverrideWithEnvVar(t *testing.T) {
	appEnv := "development"
	envVars := map[string]string{"TEMPORAL_HOSTPORT": "override:1234"}
	yamlContent := `
temporal:
hostPort: "localhost:7233"
`
	basePath := "cmd/config"
	dynamicDir := ""
	chdirTo := ""
	options := []ReadConfigOption(nil)
	expectedHostPort := "override:1234"

	tempDir := createTempConfig(t, basePath, dynamicDir, appEnv, yamlContent)

	err := os.Setenv(env.ApplicationEnvKey, appEnv)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	for k, v := range envVars {
		err := os.Setenv(k, v)
		require.NoError(t, err)
		t.Cleanup(func() { _ = os.Unsetenv(k) })
	}

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir
	if chdirTo != "" {
		chdirPath = filepath.Join(tempDir, chdirTo)
		err = os.MkdirAll(chdirPath, 0755)
		require.NoError(t, err)
	}
	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := zaptest.NewLogger(t)

	var conf TestConfig
	err = LoadConfig(&conf, logger, options...)
	require.NoError(t, err)
	require.Equal(t, expectedHostPort, conf.Temporal.HostPort)
}

// TestEnvPlaceholderReplacement tests replacing env:// placeholder with env var value.
func TestEnvPlaceholderReplacement(t *testing.T) {
	appEnv := "development"
	envVars := map[string]string{"MY_HOST": "fromenv:5678"}
	yamlContent := `
temporal:
hostPort: "env://MY_HOST"
`
	basePath := "cmd/config"
	dynamicDir := ""
	chdirTo := ""
	options := []ReadConfigOption(nil)
	expectedHostPort := "fromenv:5678"

	tempDir := createTempConfig(t, basePath, dynamicDir, appEnv, yamlContent)

	err := os.Setenv(env.ApplicationEnvKey, appEnv)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	for k, v := range envVars {
		err := os.Setenv(k, v)
		require.NoError(t, err)
		t.Cleanup(func() { _ = os.Unsetenv(k) })
	}

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir
	if chdirTo != "" {
		chdirPath = filepath.Join(tempDir, chdirTo)
		err = os.MkdirAll(chdirPath, 0755)
		require.NoError(t, err)
	}
	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := zaptest.NewLogger(t)

	var conf TestConfig
	err = LoadConfig(&conf, logger, options...)
	require.NoError(t, err)
	require.Equal(t, expectedHostPort, conf.Temporal.HostPort)
}

// TestMissingEnvPlaceholder tests handling missing env var for placeholder.
func TestMissingEnvPlaceholder(t *testing.T) {
	appEnv := "development"
	yamlContent := `
temporal:
hostPort: "env://NON_EXISTENT"
`
	basePath := "cmd/config"
	dynamicDir := ""
	chdirTo := ""
	options := []ReadConfigOption(nil)
	expectedHostPort := ""

	tempDir := createTempConfig(t, basePath, dynamicDir, appEnv, yamlContent)

	err := os.Setenv(env.ApplicationEnvKey, appEnv)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir
	if chdirTo != "" {
		chdirPath = filepath.Join(tempDir, chdirTo)
		err = os.MkdirAll(chdirPath, 0755)
		require.NoError(t, err)
	}
	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := zaptest.NewLogger(t)

	var conf TestConfig
	err = LoadConfig(&conf, logger, options...)
	require.NoError(t, err)
	require.Equal(t, expectedHostPort, conf.Temporal.HostPort)
}

// TestInvalidEnvironment tests error on invalid environment.
func TestInvalidEnvironment(t *testing.T) {
	appEnv := "invalid"
	yamlContent := `
temporal:
hostPort: "localhost:7233"
`
	basePath := "cmd/config"
	dynamicDir := ""
	chdirTo := ""
	options := []ReadConfigOption(nil)
	expectedErrStr := "invalid environment"

	tempDir := createTempConfig(t, basePath, dynamicDir, appEnv, yamlContent)

	err := os.Setenv(env.ApplicationEnvKey, appEnv)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir
	if chdirTo != "" {
		chdirPath = filepath.Join(tempDir, chdirTo)
		err = os.MkdirAll(chdirPath, 0755)
		require.NoError(t, err)
	}
	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := zaptest.NewLogger(t)

	var conf TestConfig
	err = LoadConfig(&conf, logger, options...)
	require.Error(t, err)
	require.Contains(t, err.Error(), expectedErrStr)
}

// TestMissingConfigFile tests error when config file is missing.
func TestMissingConfigFile(t *testing.T) {
	appEnv := "development"
	yamlContent := "" // No file
	basePath := "cmd/config"
	dynamicDir := ""
	chdirTo := ""
	options := []ReadConfigOption(nil)
	expectedErrStr := "failed to read configuration file"

	tempDir := createTempConfig(t, basePath, dynamicDir, appEnv, yamlContent)

	err := os.Setenv(env.ApplicationEnvKey, appEnv)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir
	if chdirTo != "" {
		chdirPath = filepath.Join(tempDir, chdirTo)
		err = os.MkdirAll(chdirPath, 0755)
		require.NoError(t, err)
	}
	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := zaptest.NewLogger(t)

	var conf TestConfig
	err = LoadConfig(&conf, logger, options...)
	require.Error(t, err)
	require.Contains(t, err.Error(), expectedErrStr)
}

// TestBinaryTargetPath tests path adjustment for binary target directory.
func TestBinaryTargetPath(t *testing.T) {
	appEnv := "development"
	yamlContent := `
temporal:
hostPort: "localhost:7233"
`
	basePath := "config" // For binary
	dynamicDir := ""
	chdirTo := "target"
	options := []ReadConfigOption(nil)
	expectedHostPort := "localhost:7233"

	tempDir := createTempConfig(t, basePath, dynamicDir, appEnv, yamlContent)

	err := os.Setenv(env.ApplicationEnvKey, appEnv)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := filepath.Join(tempDir, chdirTo)
	err = os.MkdirAll(chdirPath, 0755)
	require.NoError(t, err)
	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := zaptest.NewLogger(t)

	var conf TestConfig
	err = LoadConfig(&conf, logger, options...)
	require.NoError(t, err)
	require.Equal(t, expectedHostPort, conf.Temporal.HostPort)
}

// TestDynamicDirectory tests using the dynamic directory option.
func TestDynamicDirectory(t *testing.T) {
	appEnv := "development"
	yamlContent := `
temporal:
hostPort: "localhost:7233"
`
	basePath := "cmd/config"
	dynamicDir := "subdir"
	chdirTo := ""
	options := []ReadConfigOption{WithDynamicDir("subdir")}
	expectedHostPort := "localhost:7233"

	tempDir := createTempConfig(t, basePath, dynamicDir, appEnv, yamlContent)

	err := os.Setenv(env.ApplicationEnvKey, appEnv)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir
	if chdirTo != "" {
		chdirPath = filepath.Join(tempDir, chdirTo)
		err = os.MkdirAll(chdirPath, 0755)
		require.NoError(t, err)
	}
	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := zaptest.NewLogger(t)

	var conf TestConfig
	err = LoadConfig(&conf, logger, options...)
	require.NoError(t, err)
	require.Equal(t, expectedHostPort, conf.Temporal.HostPort)
}

// TestAbsolutePath tests using the absolute path option.
func TestAbsolutePath(t *testing.T) {
	appEnv := "development"
	yamlContent := `
temporal:
hostPort: "localhost:7233"
`
	basePath := "cmd/config"
	dynamicDir := ""
	chdirTo := ""
	expectedHostPort := "localhost:7233"

	tempDir := createTempConfig(t, basePath, dynamicDir, appEnv, yamlContent)

	configPath := filepath.Join(tempDir, basePath)
	if dynamicDir != "" {
		configPath = filepath.Join(configPath, dynamicDir)
	}
	options := []ReadConfigOption{WithAbsolutePath(configPath)}

	err := os.Setenv(env.ApplicationEnvKey, appEnv)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir
	if chdirTo != "" {
		chdirPath = filepath.Join(tempDir, chdirTo)
		err = os.MkdirAll(chdirPath, 0755)
		require.NoError(t, err)
	}
	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := zaptest.NewLogger(t)

	var conf TestConfig
	err = LoadConfig(&conf, logger, options...)
	require.NoError(t, err)
	require.Equal(t, expectedHostPort, conf.Temporal.HostPort)
}

//type TestConfig struct {
//	Temporal struct {
//		HostPort string `mapstructure:"hostPort"`
//	} `mapstructure:"temporal"`
//}
//
//func TestLoadConfig(t *testing.T) {
//	// Helper function to create temporary config directory and file
//	createTempConfig := func(t *testing.T, basePath string, dynamicDir string, env string, content string) (string, func()) {
//		t.Helper()
//		tempDir, err := os.MkdirTemp("", "config-test-*")
//		if err != nil {
//			t.Fatalf("failed to create temp dir: %v", err)
//		}
//
//		configPath := filepath.Join(tempDir, basePath)
//		if dynamicDir != "" {
//			configPath = filepath.Join(configPath, dynamicDir)
//		}
//		if err := os.MkdirAll(configPath, 0755); err != nil {
//			t.Fatalf("failed to create config dir: %v", err)
//		}
//
//		cleanup := func() {
//			os.RemoveAll(tempDir)
//		}
//
//		if content != "" {
//			filePath := filepath.Join(configPath, env+".yaml")
//			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
//				t.Fatalf("failed to write yaml file: %v", err)
//			}
//		}
//
//		return tempDir, cleanup
//	}
//
//	// Assuming env.ApplicationEnvKey is "APP_ENV"
//	// Assuming valid environments include "development"
//
//	tests := []struct {
//		name           string
//		appEnv         string
//		envVars        map[string]string // Additional env vars beyond APP_ENV
//		yamlContent    string
//		options        []ReadConfigOption
//		chdirTo        string // "" for tempDir, or "target", "app"
//		basePath       string // "cmd/config" or "config" for binary
//		dynamicDir     string
//		expectedConfig TestConfig
//		expectErr      bool
//		expectedErrStr string
//	}{
//		{
//			name:        "Basic configuration loading from YAML file",
//			appEnv:      "development",
//			envVars:     nil,
//			yamlContent: `temporal: {hostPort: "localhost:7233"}`,
//			options:     nil,
//			chdirTo:     "",
//			basePath:    "cmd/config",
//			dynamicDir:  "",
//			expectedConfig: TestConfig{Temporal: struct {
//				HostPort string `mapstructure:"hostPort"`
//			}(struct{ HostPort string }{"localhost:7233"})},
//			expectErr: false,
//		},
//		{
//			name:        "Override YAML value with environment variable",
//			appEnv:      "development",
//			envVars:     map[string]string{"TEMPORAL_HOSTPORT": "override:1234"},
//			yamlContent: `temporal: {hostPort: "localhost:7233"}`,
//			options:     nil,
//			chdirTo:     "",
//			basePath:    "cmd/config",
//			dynamicDir:  "",
//			expectedConfig: TestConfig{Temporal: struct {
//				HostPort string `mapstructure:"hostPort"`
//			}(struct{ HostPort string }{"override:1234"})},
//			expectErr: false,
//		},
//		{
//			name:        "Replace env:// placeholder with actual env var",
//			appEnv:      "development",
//			envVars:     map[string]string{"MY_HOST": "fromenv:5678"},
//			yamlContent: `temporal: {hostPort: "env://MY_HOST"}`,
//			options:     nil,
//			chdirTo:     "",
//			basePath:    "cmd/config",
//			dynamicDir:  "",
//			expectedConfig: TestConfig{Temporal: struct {
//				HostPort string `mapstructure:"hostPort"`
//			}(struct{ HostPort string }{"fromenv:5678"})},
//			expectErr: false,
//		},
//		{
//			name:        "Handle missing env var for placeholder (sets to empty)",
//			appEnv:      "development",
//			envVars:     nil,
//			yamlContent: `temporal: {hostPort: "env://NON_EXISTENT"}`,
//			options:     nil,
//			chdirTo:     "",
//			basePath:    "cmd/config",
//			dynamicDir:  "",
//			expectedConfig: TestConfig{Temporal: struct {
//				HostPort string `mapstructure:"hostPort"`
//			}(struct{ HostPort string }{""})},
//			expectErr: false,
//		},
//		{
//			name:           "Invalid environment causes error",
//			appEnv:         "invalid",
//			envVars:        nil,
//			yamlContent:    `temporal: {hostPort: "localhost:7233"}`,
//			options:        nil,
//			chdirTo:        "",
//			basePath:       "cmd/config",
//			dynamicDir:     "",
//			expectedConfig: TestConfig{},
//			expectErr:      true,
//			expectedErrStr: "invalid environment",
//		},
//		{
//			name:           "Missing config file causes read error",
//			appEnv:         "development",
//			envVars:        nil,
//			yamlContent:    "", // No file created
//			options:        nil,
//			chdirTo:        "",
//			basePath:       "cmd/config",
//			dynamicDir:     "",
//			expectedConfig: TestConfig{},
//			expectErr:      true,
//			expectedErrStr: "failed to read configuration file",
//		},
//		{
//			name:        "Adjust path when running from binary target directory",
//			appEnv:      "development",
//			envVars:     nil,
//			yamlContent: `temporal: {hostPort: "localhost:7233"}`,
//			options:     nil,
//			chdirTo:     "target",
//			basePath:    "config",
//			dynamicDir:  "",
//			expectedConfig: TestConfig{Temporal: struct {
//				HostPort string `mapstructure:"hostPort"`
//			}(struct{ HostPort string }{"localhost:7233"})},
//			expectErr: false,
//		},
//		{
//			name:        "Use dynamic directory option",
//			appEnv:      "development",
//			envVars:     nil,
//			yamlContent: `temporal: {hostPort: "localhost:7233"}`,
//			options:     []ReadConfigOption{WithDynamicDir("subdir")},
//			chdirTo:     "",
//			basePath:    "cmd/config",
//			dynamicDir:  "subdir",
//			expectedConfig: TestConfig{Temporal: struct {
//				HostPort string `mapstructure:"hostPort"`
//			}(struct{ HostPort string }{"localhost:7233"})},
//			expectErr: false,
//		},
//		{
//			name:        "Use absolute path option",
//			appEnv:      "development",
//			envVars:     nil,
//			yamlContent: `temporal: {hostPort: "localhost:7233"}`,
//			options:     []ReadConfigOption{WithAbsolutePath("ABSOLUTE_PATH_PLACEHOLDER")},
//			chdirTo:     "",
//			basePath:    "cmd/config",
//			dynamicDir:  "",
//			expectedConfig: TestConfig{Temporal: struct {
//				HostPort string `mapstructure:"hostPort"`
//			}(struct{ HostPort string }{"localhost:7233"})},
//			expectErr: false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Create temp config structure
//			tempDir, cleanup := createTempConfig(t, tt.basePath, tt.dynamicDir, tt.appEnv, tt.yamlContent)
//			defer cleanup()
//
//			// Set APP_ENV
//			os.Setenv(env.ApplicationEnvKey, tt.appEnv)
//			defer os.Unsetenv(env.ApplicationEnvKey)
//
//			// Set additional env vars
//			for k, v := range tt.envVars {
//				os.Setenv(k, v)
//			}
//			defer func() {
//				for k := range tt.envVars {
//					os.Unsetenv(k)
//				}
//			}()
//
//			// Change working directory
//			oldWd, err := os.Getwd()
//			if err != nil {
//				t.Fatalf("failed to get current wd: %v", err)
//			}
//			defer os.Chdir(oldWd)
//
//			chdirPath := tempDir
//			if tt.chdirTo != "" {
//				chdirPath = filepath.Join(tempDir, tt.chdirTo)
//				if err := os.MkdirAll(chdirPath, 0755); err != nil {
//					t.Fatalf("failed to create chdir path: %v", err)
//				}
//			}
//			if err := os.Chdir(chdirPath); err != nil {
//				t.Fatalf("failed to chdir: %v", err)
//			}
//
//			// Replace placeholder in absolute path if present
//			options := tt.options
//			if len(options) > 0 {
//				configPath := filepath.Join(tempDir, tt.basePath)
//				if tt.dynamicDir != "" {
//					configPath = filepath.Join(configPath, tt.dynamicDir)
//				}
//				for i, opt := range options {
//					// Hack to check if it's WithAbsolutePath
//					if fmt.Sprintf("%v", opt) == "ABSOLUTE_PATH_PLACEHOLDER" {
//						options[i] = WithAbsolutePath(configPath)
//					}
//				}
//			}
//
//			// Reset viper
//			viper.Reset()
//
//			// Test logger
//			logger := zaptest.NewLogger(t)
//
//			// Execute LoadConfig
//			var conf TestConfig
//			err = LoadConfig(&conf, logger, options...)
//
//			// Validate error
//			if tt.expectErr {
//				if err == nil {
//					t.Errorf("expected an error but got none")
//				} else if tt.expectedErrStr != "" && !strings.Contains(err.Error(), tt.expectedErrStr) {
//					t.Errorf("expected error containing %q, got: %v", tt.expectedErrStr, err)
//				}
//			} else {
//				if err != nil {
//					t.Errorf("unexpected error: %v", err)
//				}
//				if conf.Temporal.HostPort != tt.expectedConfig.Temporal.HostPort {
//					t.Errorf("expected HostPort %q, got %q", tt.expectedConfig.Temporal.HostPort, conf.Temporal.HostPort)
//				}
//			}
//		})
//	}
//}
