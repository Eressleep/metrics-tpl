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
	memStorage := DataStorage{
		make(map[string]float64),
		make(map[string]int64),
	}

	fmt.Println("Starting server...")
	mux := http.NewServeMux()

	mux.HandleFunc("/update/", updateHandler(&memStorage))

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

		if r.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, "Invalid data format", http.StatusNotFound)
		}

		if len(path) < 4 {
			http.Error(w, "Invalid URL format", http.StatusNotFound)
			return
		}

		metricType, metricName, metricValue := path[2], path[3], path[4]

		if metricName == "" {
			http.Error(w, "Metric name is required", http.StatusNotFound)
			return
		}

		// Проверяем наличие значения
		if len(path) < 5 {
			http.Error(w, "Metric value is required", http.StatusNotFound)
			return
		}

		// Проверка на пустое значение
		if metricValue == "" {
			http.Error(w, "Metric value cannot be empty", http.StatusNotFound)
			return
		}

		switch metricType {
		case "counter":
			pathValue, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				badRequest(w)
				return
			}
			memStorage.counters[metricName] += pathValue
			w.WriteHeader(http.StatusOK)
		case "gauge":
			pathValue, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				badRequest(w)
				return
			}
			memStorage.gauges[metricName] = pathValue
		default:
			badRequest(w)
		}
	}

}

func badRequest(w http.ResponseWriter) {
	http.Error(w, "Bad request", http.StatusBadRequest)
}
