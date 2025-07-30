package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/rainbow-me/platform-tools/common/config"
	"github.com/rainbow-me/platform-tools/common/env"
	"github.com/rainbow-me/platform-tools/common/test"
)

type TestConfig struct {
	Temporal struct {
		HostPort string `mapstructure:"hostPort"`
	} `mapstructure:"temporal"`
}

// createTempConfig creates a temporary config directory and file, registering cleanup.
func createTempConfig(t *testing.T, dynamicDir, appEnv, content string) string {
	t.Helper()
	tempDir := t.TempDir()
	basePath := "cmd/config"
	configPath := filepath.Join(tempDir, basePath)
	if dynamicDir != "" {
		configPath = filepath.Join(configPath, dynamicDir)
	}
	err := os.MkdirAll(configPath, 0755)
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
	yamlContent := `temporal: {hostPort: "localhost:7233"}`
	dynamicDir := ""
	options := []config.ReadConfigOption(nil)
	expectedHostPort := "localhost:7233"

	tempDir := createTempConfig(t, dynamicDir, appEnv, yamlContent)

	t.Setenv(env.ApplicationEnvKey, appEnv)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir

	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := test.NewLogger(t)

	var conf TestConfig
	err = config.LoadConfig(&conf, logger, options...)
	require.NoError(t, err)
	require.Equal(t, expectedHostPort, conf.Temporal.HostPort)
}

// TestOverrideWithEnvVar tests overriding a YAML value with an environment variable.
func TestOverrideWithEnvVar(t *testing.T) {
	appEnv := "development"
	envVars := map[string]string{"TEMPORAL_HOSTPORT": "override:1234"}
	yamlContent := `temporal: {hostPort: "localhost:7233"}`
	dynamicDir := ""
	options := []config.ReadConfigOption(nil)
	expectedHostPort := "override:1234"

	tempDir := createTempConfig(t, dynamicDir, appEnv, yamlContent)

	t.Setenv(env.ApplicationEnvKey, appEnv)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	for k, v := range envVars {
		t.Setenv(k, v)
		t.Cleanup(func() { _ = os.Unsetenv(k) })
	}

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir
	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := test.NewLogger(t)

	var conf TestConfig
	err = config.LoadConfig(&conf, logger, options...)
	require.NoError(t, err)
	require.Equal(t, expectedHostPort, conf.Temporal.HostPort)
}

// TestEnvPlaceholderReplacement tests replacing env:// placeholder with env var value.
func TestEnvPlaceholderReplacement(t *testing.T) {
	appEnv := "development"
	envVars := map[string]string{"MY_HOST": "fromenv:5678"}
	yamlContent := `temporal: {hostPort: "env://MY_HOST"}`
	dynamicDir := ""
	options := []config.ReadConfigOption(nil)
	expectedHostPort := "fromenv:5678"

	tempDir := createTempConfig(t, dynamicDir, appEnv, yamlContent)

	t.Setenv(env.ApplicationEnvKey, appEnv)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	for k, v := range envVars {
		t.Setenv(k, v)
		t.Cleanup(func() { _ = os.Unsetenv(k) })
	}

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir

	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := test.NewLogger(t)

	var conf TestConfig
	err = config.LoadConfig(&conf, logger, options...)
	require.NoError(t, err)
	require.Equal(t, expectedHostPort, conf.Temporal.HostPort)
}

// TestMissingEnvPlaceholder tests handling missing env var for placeholder.
func TestMissingEnvPlaceholder(t *testing.T) {
	appEnv := "development"
	yamlContent := `temporal: {hostPort: "env://NON_EXISTENT"}`
	dynamicDir := ""
	options := []config.ReadConfigOption(nil)
	expectedHostPort := ""

	tempDir := createTempConfig(t, dynamicDir, appEnv, yamlContent)

	t.Setenv(env.ApplicationEnvKey, appEnv)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir

	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := test.NewLogger(t)

	var conf TestConfig
	err = config.LoadConfig(&conf, logger, options...)
	require.NoError(t, err)
	require.Equal(t, expectedHostPort, conf.Temporal.HostPort)
}

// TestInvalidEnvironment tests error on invalid environment.
func TestInvalidEnvironment(t *testing.T) {
	appEnv := "invalid"
	yamlContent := `temporal: {hostPort: "localhost:7233"}`
	dynamicDir := ""
	options := []config.ReadConfigOption(nil)
	expectedErrStr := "invalid environment"

	tempDir := createTempConfig(t, dynamicDir, appEnv, yamlContent)

	t.Setenv(env.ApplicationEnvKey, appEnv)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir

	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := test.NewLogger(t)

	var conf TestConfig
	err = config.LoadConfig(&conf, logger, options...)
	require.Error(t, err)
	require.Contains(t, err.Error(), expectedErrStr)
}

// TestMissingConfigFile tests error when config file is missing.
func TestMissingConfigFile(t *testing.T) {
	appEnv := "development"
	yamlContent := "" // No file
	dynamicDir := ""
	options := []config.ReadConfigOption(nil)
	expectedErrStr := "failed to read configuration file"

	tempDir := createTempConfig(t, dynamicDir, appEnv, yamlContent)

	t.Setenv(env.ApplicationEnvKey, appEnv)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir

	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := test.NewLogger(t)

	var conf TestConfig
	err = config.LoadConfig(&conf, logger, options...)
	require.Error(t, err)
	require.Contains(t, err.Error(), expectedErrStr)
}

// TestDynamicDirectory tests using the dynamic directory option.
func TestDynamicDirectory(t *testing.T) {
	appEnv := "development"
	yamlContent := `temporal: {hostPort: "localhost:7233"}`
	dynamicDir := "subdir"
	options := []config.ReadConfigOption{config.WithDynamicDir("subdir")}
	expectedHostPort := "localhost:7233"

	tempDir := createTempConfig(t, dynamicDir, appEnv, yamlContent)

	t.Setenv(env.ApplicationEnvKey, appEnv)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir

	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := test.NewLogger(t)

	var conf TestConfig
	err = config.LoadConfig(&conf, logger, options...)
	require.NoError(t, err)
	require.Equal(t, expectedHostPort, conf.Temporal.HostPort)
}

// TestAbsolutePath tests using the absolute path option.
func TestAbsolutePath(t *testing.T) {
	appEnv := "development"
	yamlContent := `temporal: {hostPort: "localhost:7233"}`
	basePath := "cmd/config"
	dynamicDir := ""
	expectedHostPort := "localhost:7233"

	tempDir := createTempConfig(t, dynamicDir, appEnv, yamlContent)

	configPath := filepath.Join(tempDir, basePath)

	options := []config.ReadConfigOption{config.WithAbsolutePath(configPath)}

	t.Setenv(env.ApplicationEnvKey, appEnv)
	t.Cleanup(func() { _ = os.Unsetenv(env.ApplicationEnvKey) })

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	chdirPath := tempDir

	err = os.Chdir(chdirPath)
	require.NoError(t, err)

	viper.Reset()

	logger := test.NewLogger(t)

	var conf TestConfig
	err = config.LoadConfig(&conf, logger, options...)
	require.NoError(t, err)
	require.Equal(t, expectedHostPort, conf.Temporal.HostPort)
}
