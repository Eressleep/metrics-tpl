// Package build содержит информацию о версии сборки.
//
// Переменные устанавливаются при сборке через ldflags:
//
//	go build -ldflags "-X 'github.com/Eressleep/metrics-tpl/internal/build.Version=1.0.0' \
//	                   -X 'github.com/Eressleep/metrics-tpl/internal/build.Date=2024-01-01' \
//	                   -X 'github.com/Eressleep/metrics-tpl/internal/build.Commit=abc123'" \
//	           ./cmd/server/
package build

import "fmt"

var (
	// Version - версия сборки
	Version = "N/A"
	// Date - дата сборки
	Date = "N/A"
	// Commit - коммит сборки
	Commit = "N/A"
)

// PrintBuildInfo выводит информацию о сборке в stdout
func PrintBuildInfo() {
	fmt.Printf("Build version: %s\n", Version)
	fmt.Printf("Build date: %s\n", Date)
	fmt.Printf("Build commit: %s\n", Commit)
}
