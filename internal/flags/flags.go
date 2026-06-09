package flags

import (
	"flag"
	"os"
	"strconv"
	"time"
)

// IsFlagSet checks if a flag was explicitly set via command line
func IsFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// GetStringFromEnv returns string value from environment variable
func GetStringFromEnv(envName, defaultValue string) string {
	if value := os.Getenv(envName); value != "" {
		return value
	}
	return defaultValue
}

// GetIntFromEnv returns int value from environment variable
func GetIntFromEnv(envName string, defaultValue int) int {
	if value := os.Getenv(envName); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetBoolFromEnv returns bool value from environment variable
func GetBoolFromEnv(envName string, defaultValue bool) bool {
	if value := os.Getenv(envName); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// GetDurationFromEnv returns time.Duration value from environment variable
func GetDurationFromEnv(envName string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(envName); value != "" {
		if dur, err := time.ParseDuration(value); err == nil {
			return dur
		}
	}
	return defaultValue
}

// GetConfigString returns string config value with proper priority:
// 1. Command line flag (if explicitly set)
// 2. Environment variable
// 3. Default value
func GetConfigString(flagValue *string, flagName, envName, defaultValue string) string {
	// Priority 1: Command line flag (only if explicitly set)
	if IsFlagSet(flagName) {
		return *flagValue
	}

	// Priority 2: Environment variable
	if envValue := os.Getenv(envName); envValue != "" {
		return envValue
	}

	// Priority 3: Default value
	return defaultValue
}

// GetConfigInt returns int config value with proper priority:
// 1. Command line flag (if explicitly set)
// 2. Environment variable
// 3. Default value
func GetConfigInt(flagValue *int, flagName, envName string, defaultValue int) int {
	// Priority 1: Command line flag (only if explicitly set)
	if IsFlagSet(flagName) {
		return *flagValue
	}

	// Priority 2: Environment variable
	if envValue := os.Getenv(envName); envValue != "" {
		if intVal, err := strconv.Atoi(envValue); err == nil {
			return intVal
		}
	}

	// Priority 3: Default value
	return defaultValue
}

// GetConfigBool returns bool config value with proper priority:
// 1. Command line flag (if explicitly set)
// 2. Environment variable
// 3. Default value
func GetConfigBool(flagValue *bool, flagName, envName string, defaultValue bool) bool {
	// Priority 1: Command line flag (only if explicitly set)
	if IsFlagSet(flagName) {
		return *flagValue
	}

	// Priority 2: Environment variable
	if envValue := os.Getenv(envName); envValue != "" {
		if boolVal, err := strconv.ParseBool(envValue); err == nil {
			return boolVal
		}
	}

	// Priority 3: Default value
	return defaultValue
}

// GetConfigDuration returns time.Duration config value with proper priority:
// 1. Command line flag (if explicitly set)
// 2. Environment variable
// 3. Default value
func GetConfigDuration(flagValue *time.Duration, flagName, envName string, defaultValue time.Duration) time.Duration {
	// Priority 1: Command line flag (only if explicitly set)
	if IsFlagSet(flagName) {
		return *flagValue
	}

	// Priority 2: Environment variable
	if envValue := os.Getenv(envName); envValue != "" {
		if dur, err := time.ParseDuration(envValue); err == nil {
			return dur
		}
	}

	// Priority 3: Default value
	return defaultValue
}
