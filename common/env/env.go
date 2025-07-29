package env

import (
	"fmt"
	"slices"
	"strings"
)

const ApplicationEnvKey = "ENVIRONMENT"

// Environment represents the application deployment environment
type Environment string

const (
	EnvironmentLocal       Environment = "local"
	EnvironmentLocalDocker Environment = "local-docker"
	EnvironmentDevelopment Environment = "development"
	EnvironmentStaging     Environment = "staging"
	EnvironmentProduction  Environment = "production"
)

func (e Environment) String() string { return string(e) }

func IsEnvironmentValid(environment string) error {
	supportedEnv := []string{
		EnvironmentLocal.String(),
		EnvironmentLocalDocker.String(),
		EnvironmentDevelopment.String(),
		EnvironmentStaging.String(),
		EnvironmentProduction.String(),
	}
	if slices.Contains(supportedEnv, environment) {
		return nil
	}

	envList := strings.Join(supportedEnv, ", ")

	return fmt.Errorf("invalid environment: %s must be set to one of %s", ApplicationEnvKey, envList)
}

func FromString(environment string) (Environment, error) {
	env := Environment(environment)
	if err := IsEnvironmentValid(environment); err != nil {
		return "", err
	}
	return env, nil
}
