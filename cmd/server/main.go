package main

import (
	"fmt"
	"net/http"
	"strconv"
)

type gauge float64
type counter int64

type MemStorage struct {
	Gauge   gauge
	Counter map[string]counter
}

type Storage interface {
	Add(Type string, Name string, Val string)
}

var storage = MemStorage{
	Counter: make(map[string]counter),
}

func (stor *MemStorage) Add(Type string, Name string, Val string) {
	switch Type {
	case "gauge":
		val, _ := strconv.ParseFloat(Val, 64)
		stor.Gauge = gauge(val)
	case "counter":
		val, _ := strconv.ParseInt(Val, 10, 64)
		stor.Counter[Name] += counter(val)
	}
}

func mainPage(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "", http.StatusMethodNotAllowed)
		return
	}

	typeMetric := req.PathValue("type")
	nameMetric := req.PathValue("name")
	valMetric := req.PathValue("val")

	if nameMetric == "" {
		http.Error(res, "", http.StatusNotFound)
		return
	}

	if typeMetric != "gauge" && typeMetric != "counter" {
		http.Error(res, "", http.StatusBadRequest)
		return
	}

	switch typeMetric {
	case "gauge":
		_, err := strconv.ParseFloat(valMetric, 64)
		if err != nil {
			http.Error(res, "", http.StatusBadRequest)
			return
		}
	case "counter":
		_, err := strconv.ParseInt(valMetric, 10, 64)
		if err != nil {
			http.Error(res, "", http.StatusBadRequest)
			return
		}
	}

	storage.Add(typeMetric, nameMetric, valMetric)

	fmt.Println("counter", storage.Counter)
	fmt.Println("gauge", storage.Gauge)

	res.WriteHeader(http.StatusOK)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(`/update/{type}/{name}/{val}`, mainPage)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
