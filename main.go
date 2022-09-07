package main

import (
	"net/http"
	"os"
)

const (
	dataFilePath = "data.json"
)

func Mux() *http.ServeMux {
	_, err := os.Stat(dataFilePath)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(dataFilePath, []byte("{}"), 0666); err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	return mux
}

func main() {
	mux := Mux()
	http.ListenAndServe(":9080", mux)
}
