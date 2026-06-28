package flags

import (
	"flag"
	"os"
	"strconv"
	"time"
)

// ConfigFileFlagName имя флага для файла конфигурации
const ConfigFileFlagName = "config"

// IsFlagSet проверяет, был ли установлен флаг
func IsFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// GetStringFromEnv получает строку из переменной окружения
func GetStringFromEnv(envName, defaultValue string) string {
	if value := os.Getenv(envName); value != "" {
		return value
	}
	return defaultValue
}

// GetIntFromEnv получает int из переменной окружения
func GetIntFromEnv(envName string, defaultValue int) int {
	if value := os.Getenv(envName); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetBoolFromEnv получает bool из переменной окружения
func GetBoolFromEnv(envName string, defaultValue bool) bool {
	if value := os.Getenv(envName); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// GetDurationFromEnv получает duration из переменной окружения
func GetDurationFromEnv(envName string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(envName); value != "" {
		if dur, err := time.ParseDuration(value); err == nil {
			return dur
		}
	}
	return defaultValue
}

// GetConfigString получает строку с приоритетом: флаг > env > default
func GetConfigString(flagValue *string, flagName, envName, defaultValue string) string {
	if IsFlagSet(flagName) {
		return *flagValue
	}

	if envValue := os.Getenv(envName); envValue != "" {
		return envValue
	}

	return defaultValue
}

// GetConfigStringWithFile получает строку с приоритетом: флаг > env > file > default
func GetConfigStringWithFile(flagValue *string, flagName, envName, fileValue, defaultValue string) string {
	if IsFlagSet(flagName) {
		return *flagValue
	}

	if envValue := os.Getenv(envName); envValue != "" {
		return envValue
	}

	if fileValue != "" {
		return fileValue
	}

	return defaultValue
}

// GetConfigInt получает int с приоритетом: флаг > env > default
func GetConfigInt(flagValue *int, flagName, envName string, defaultValue int) int {
	if IsFlagSet(flagName) {
		return *flagValue
	}

	if envValue := os.Getenv(envName); envValue != "" {
		if intVal, err := strconv.Atoi(envValue); err == nil {
			return intVal
		}
	}

	return defaultValue
}

// GetConfigIntWithFile получает int с приоритетом: флаг > env > file > default
func GetConfigIntWithFile(flagValue *int, flagName, envName string, fileValue int, defaultValue int) int {
	if IsFlagSet(flagName) {
		return *flagValue
	}

	if envValue := os.Getenv(envName); envValue != "" {
		if intVal, err := strconv.Atoi(envValue); err == nil {
			return intVal
		}
	}

	if fileValue != 0 {
		return fileValue
	}

	return defaultValue
}

// GetConfigBool получает bool с приоритетом: флаг > env > default
func GetConfigBool(flagValue *bool, flagName, envName string, defaultValue bool) bool {
	if IsFlagSet(flagName) {
		return *flagValue
	}

	if envValue := os.Getenv(envName); envValue != "" {
		if boolVal, err := strconv.ParseBool(envValue); err == nil {
			return boolVal
		}
	}

	return defaultValue
}

// GetConfigBoolWithFile получает bool с приоритетом: флаг > env > file > default
func GetConfigBoolWithFile(flagValue *bool, flagName, envName string, fileValue *bool, defaultValue bool) bool {
	if IsFlagSet(flagName) {
		return *flagValue
	}

	if envValue := os.Getenv(envName); envValue != "" {
		if boolVal, err := strconv.ParseBool(envValue); err == nil {
			return boolVal
		}
	}

	if fileValue != nil {
		return *fileValue
	}

	return defaultValue
}

// GetConfigDuration получает duration с приоритетом: флаг > env > default
func GetConfigDuration(flagValue *time.Duration, flagName, envName string, defaultValue time.Duration) time.Duration {
	if IsFlagSet(flagName) {
		return *flagValue
	}

	if envValue := os.Getenv(envName); envValue != "" {
		if dur, err := time.ParseDuration(envValue); err == nil {
			return dur
		}
	}

	return defaultValue
}

// GetConfigDurationWithFile получает duration с приоритетом: флаг > env > file > default
func GetConfigDurationWithFile(flagValue *time.Duration, flagName, envName, fileValue string, defaultValue time.Duration) time.Duration {
	if IsFlagSet(flagName) {
		return *flagValue
	}

	if envValue := os.Getenv(envName); envValue != "" {
		if dur, err := time.ParseDuration(envValue); err == nil {
			return dur
		}
	}

	if fileValue != "" {
		if dur, err := time.ParseDuration(fileValue); err == nil {
			return dur
		}
	}

	return defaultValue
}
