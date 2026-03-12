package flags

import (
	"flag"
	"os"
	"testing"
)

func TestIsFlagSet(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	if IsFlagSet("test") {
		t.Error("IsFlagSet should return false for unset flag")
	}

	var testFlag string
	flag.StringVar(&testFlag, "test", "default", "test flag")
	flag.Parse()

	os.Args = []string{"cmd", "-test=value"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flag.StringVar(&testFlag, "test", "default", "test flag")
	flag.Parse()

	if !IsFlagSet("test") {
		t.Error("IsFlagSet should return true for set flag")
	}
}

func TestGetStringFromEnv(t *testing.T) {
	original := os.Getenv("TEST_ENV")
	defer os.Setenv("TEST_ENV", original)

	os.Setenv("TEST_ENV", "test_value")
	result := GetStringFromEnv("TEST_ENV", "default")
	if result != "test_value" {
		t.Errorf("Expected test_value, got %s", result)
	}

	os.Unsetenv("TEST_ENV")
	result = GetStringFromEnv("TEST_ENV", "default")
	if result != "default" {
		t.Errorf("Expected default, got %s", result)
	}
}

func TestGetIntFromEnv(t *testing.T) {
	original := os.Getenv("TEST_INT")
	defer os.Setenv("TEST_INT", original)

	os.Setenv("TEST_INT", "42")
	result := GetIntFromEnv("TEST_INT", 10)
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}

	os.Setenv("TEST_INT", "not_a_number")
	result = GetIntFromEnv("TEST_INT", 10)
	if result != 10 {
		t.Errorf("Expected default 10, got %d", result)
	}

	os.Unsetenv("TEST_INT")
	result = GetIntFromEnv("TEST_INT", 10)
	if result != 10 {
		t.Errorf("Expected default 10, got %d", result)
	}
}
