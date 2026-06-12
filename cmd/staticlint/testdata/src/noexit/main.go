package main

import "os"

func main() {
	os.Exit(1) // want "прямой вызов os.Exit в функции main пакета main запрещен"
}

func helper() {
	os.Exit(0) // разрешено, не в main
}
