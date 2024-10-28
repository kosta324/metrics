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

	type_metric := req.PathValue("type")
	name_metric := req.PathValue("name")
	val_metric := req.PathValue("val")

	if name_metric == "" {
		http.Error(res, "", http.StatusNotFound)
		return
	}

	if type_metric != "gauge" && type_metric != "counter" {
		http.Error(res, "", http.StatusBadRequest)
		return
	}

	switch type_metric {
	case "gauge":
		_, err := strconv.ParseFloat(val_metric, 64)
		if err != nil {
			http.Error(res, "", http.StatusBadRequest)
			return
		}
	case "counter":
		_, err := strconv.ParseInt(val_metric, 10, 64)
		if err != nil {
			http.Error(res, "", http.StatusBadRequest)
			return
		}
	}

	storage.Add(type_metric, name_metric, val_metric)

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
