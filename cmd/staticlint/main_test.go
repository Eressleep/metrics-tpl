package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestNoExitAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), NoExitAnalyzer, "./...")
}
