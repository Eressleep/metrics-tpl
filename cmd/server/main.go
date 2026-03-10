package main

import (
	"fmt"
	"net/http"
	"strings"
)

type DataStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func main() {
	storage := DataStorage{make(map[string]float64), make(map[string]int64)}

	fmt.Println("Starting server...")
	mux := http.NewServeMux()

	mux.HandleFunc("/update/", updateHandler(storage))
}

func updateHandler(storage DataStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
		path := strings.Split(r.URL.Path, "/")
		fmt.Println(path)
	}

}
