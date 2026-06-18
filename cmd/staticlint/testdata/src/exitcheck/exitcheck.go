package exitcheck

import (
	"log"
	"os"
)

func NormalFunction() {
	os.Exit(1)
	panic("error")
	log.Fatal("error")
	log.Panic("error")
}

func WithGoroutine() {
	go func() {
		panic("error in goroutine")
	}()
}

func WithDefer() {
	defer func() {
		panic("error in defer")
	}()
}
