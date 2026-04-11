package flags

import (
	"flag"
	"os"
	"testing"
)

func setupTest() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestIsFlagSet(t *testing.T) {
	setupTest()

	if IsFlagSet("test") {
		t.Error("IsFlagSet should return false for unset flag")
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "-test=value"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	testFlag := flag.String("test", "default", "test flag")
	flag.Parse()

	if !IsFlagSet("test") {
		t.Error("IsFlagSet should return true for set flag")
	}

	if *testFlag != "value" {
		t.Errorf("Expected flag value 'value', got '%s'", *testFlag)
	}
}

func TestGetStringFromEnv(t *testing.T) {
	setupTest()

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
	setupTest()

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

func TestGetBoolFromEnv(t *testing.T) {
	setupTest()

	original := os.Getenv("TEST_BOOL")
	defer os.Setenv("TEST_BOOL", original)

	os.Setenv("TEST_BOOL", "true")
	result := GetBoolFromEnv("TEST_BOOL", false)
	if result != true {
		t.Errorf("Expected true, got %v", result)
	}

	os.Setenv("TEST_BOOL", "false")
	result = GetBoolFromEnv("TEST_BOOL", true)
	if result != false {
		t.Errorf("Expected false, got %v", result)
	}

	os.Setenv("TEST_BOOL", "invalid")
	result = GetBoolFromEnv("TEST_BOOL", true)
	if result != true {
		t.Errorf("Expected default true, got %v", result)
	}

	os.Unsetenv("TEST_BOOL")
	result = GetBoolFromEnv("TEST_BOOL", true)
	if result != true {
		t.Errorf("Expected default true, got %v", result)
	}
}

func TestGetConfigString(t *testing.T) {
	setupTest()

	oldEnv := os.Getenv("TEST_CONFIG")
	defer os.Setenv("TEST_CONFIG", oldEnv)

	os.Args = []string{"cmd", "-test-config=flag_value"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	testFlag := flag.String("test-config", "default", "test config")
	flag.Parse()

	result := GetConfigString(testFlag, "test-config", "TEST_CONFIG", "default")
	if result != "flag_value" {
		t.Errorf("Expected flag_value, got %s", result)
	}

	os.Args = []string{"cmd"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	testFlag = flag.String("test-config", "default", "test config")
	flag.Parse()

	os.Setenv("TEST_CONFIG", "env_value")
	result = GetConfigString(testFlag, "test-config", "TEST_CONFIG", "default")
	if result != "env_value" {
		t.Errorf("Expected env_value, got %s", result)
	}

	os.Unsetenv("TEST_CONFIG")
	result = GetConfigString(testFlag, "test-config", "TEST_CONFIG", "default")
	if result != "default" {
		t.Errorf("Expected default, got %s", result)
	}
}

func TestGetConfigInt(t *testing.T) {
	setupTest()

	oldEnv := os.Getenv("TEST_INT_CONFIG")
	defer os.Setenv("TEST_INT_CONFIG", oldEnv)

	os.Args = []string{"cmd", "-test-int=42"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	testFlag := flag.Int("test-int", 10, "test int config")
	flag.Parse()

	result := GetConfigInt(testFlag, "test-int", "TEST_INT_CONFIG", 10)
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}

	os.Args = []string{"cmd"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	testFlag = flag.Int("test-int", 10, "test int config")
	flag.Parse()

	os.Setenv("TEST_INT_CONFIG", "24")
	result = GetConfigInt(testFlag, "test-int", "TEST_INT_CONFIG", 10)
	if result != 24 {
		t.Errorf("Expected 24, got %d", result)
	}

	os.Unsetenv("TEST_INT_CONFIG")
	result = GetConfigInt(testFlag, "test-int", "TEST_INT_CONFIG", 10)
	if result != 10 {
		t.Errorf("Expected default 10, got %d", result)
	}
}

func TestGetConfigBool(t *testing.T) {
	setupTest()

	oldEnv := os.Getenv("TEST_BOOL_CONFIG")
	defer os.Setenv("TEST_BOOL_CONFIG", oldEnv)

	os.Args = []string{"cmd", "-test-bool=true"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	testFlag := flag.Bool("test-bool", false, "test bool config")
	flag.Parse()

	result := GetConfigBool(testFlag, "test-bool", "TEST_BOOL_CONFIG", false)
	if result != true {
		t.Errorf("Expected true, got %v", result)
	}

	os.Args = []string{"cmd"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	testFlag = flag.Bool("test-bool", false, "test bool config")
	flag.Parse()

	os.Setenv("TEST_BOOL_CONFIG", "true")
	result = GetConfigBool(testFlag, "test-bool", "TEST_BOOL_CONFIG", false)
	if result != true {
		t.Errorf("Expected true, got %v", result)
	}

	os.Unsetenv("TEST_BOOL_CONFIG")
	result = GetConfigBool(testFlag, "test-bool", "TEST_BOOL_CONFIG", true)
	if result != true {
		t.Errorf("Expected default true, got %v", result)
	}
}
