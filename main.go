package main

import (
	"fmt"
	"net/http"
	"strconv"
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

	err := http.ListenAndServe(":8080", mux)

	if err != nil {
		panic(err)
	}
}

func updateHandler(storage DataStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
		path := strings.Split(r.URL.Path, "/")
		fmt.Println(path)

		if r.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, "Invalid data format", http.StatusNotFound)
		}

		metricType, metricName, metricValue := path[2], path[3], path[4]

		switch metricType {
		case "counter":
			pathValue, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				badRequest(w)
				return
			}
			storage.counters[metricName] += pathValue
			w.WriteHeader(http.StatusOK)
		case "gauge":
			pathValue, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				badRequest(w)
				return
			}
			storage.gauges[metricName] = pathValue
		default:
			badRequest(w)
		}
	}

}

func badRequest(w http.ResponseWriter) {
	http.Error(w, "Bad request", http.StatusBadRequest)
}
