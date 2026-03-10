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
	memStorage := &DataStorage{ // Можно сразу взять указатель
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}

	fmt.Println("Starting server on :8080...")
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", updateHandler(memStorage))

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		panic(err)
	}
}

func updateHandler(memStorage *DataStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		path := strings.Split(r.URL.Path, "/")

		contentType := r.Header.Get("Content-Type")
		if contentType != "" && contentType != "text/plain" {
			http.Error(w, "Invalid data format", http.StatusNotFound)
			return
		}

		if len(path) < 4 {
			http.Error(w, "Invalid URL format", http.StatusNotFound)
			return
		}

		metricType := path[2]
		metricName := path[3]

		if metricName == "" {
			http.Error(w, "Metric name is required", http.StatusNotFound)
			return
		}

		if len(path) < 5 {
			http.Error(w, "Metric value is required", http.StatusNotFound)
			return
		}
		metricValue := path[4]

		if metricValue == "" {
			http.Error(w, "Metric value cannot be empty", http.StatusNotFound)
			return
		}

		switch metricType {
		case "counter":
			value, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				http.Error(w, "Invalid counter value", http.StatusBadRequest)
				return
			}
			memStorage.counters[metricName] += value
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK")

		case "gauge":
			value, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				http.Error(w, "Invalid gauge value", http.StatusBadRequest)
				return
			}
			memStorage.gauges[metricName] = value
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK")

		default:
			http.Error(w, "Unknown metric type", http.StatusBadRequest)
		}
	}
}
