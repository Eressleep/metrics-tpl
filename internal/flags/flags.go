package flags

import (
	"flag"
	"os"
	"strconv"
)

func IsFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func GetStringFromEnv(envName, defaultValue string) string {
	if value := os.Getenv(envName); value != "" {
		return value
	}
	return defaultValue
}

func GetIntFromEnv(envName string, defaultValue int) int {
	if value := os.Getenv(envName); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func GetConfigString(flagValue *string, flagName, envName, defaultValue string) string {
	if IsFlagSet(flagName) {
		return *flagValue
	}

	if envValue := os.Getenv(envName); envValue != "" {
		return envValue
	}

	return defaultValue
}

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
