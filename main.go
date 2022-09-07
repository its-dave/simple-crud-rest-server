package main

import (
	"errors"
	"net/http"
	"os"
	"strings"
)

const (
	dataFilePath = "data.json"
)

func Mux() *http.ServeMux {
	// Ensure data file exists
	_, err := os.Stat(dataFilePath)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(dataFilePath, []byte("{}"), 0666); err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		url := strings.TrimPrefix(r.URL.Path, "/")
		url = strings.TrimSuffix(url, "/")
		urlParts := strings.Split(url, "/")
		switch len(urlParts) {
		case 1:
			if r.Method != http.MethodPost {
				// TODO: 405
			}
			// TODO: if key already has a value: 400
			// TODO: append {"event":"create","value":value} to key array: 202
		case 2:
			// TODO: /api/key is called, must not be POST
			switch r.Method {
			case http.MethodGet:
				// TODO: if key doesn't exist: 404
				// TODO: if key has no value in final array entry: 204
				// TODO: print value in final array entry to w
			case http.MethodPatch, http.MethodPut:
				// TODO: if key doesn't exist: 404
				// TODO: if key has no value in final array entry: 400
				// TODO: append {"event":"update","value":value} to key array: 204
			case http.MethodDelete:
				// TODO: if key doesn't exist: 404
				// TODO: if key has no value in final array entry: 400
				// TODO: append {"event":"delete"} to key array: 204
			default:
				// TODO: 405
			}
		case 3:
			if urlParts[2] != "history" {
				// TODO: 404
			}
			// TODO: if key doesn't exist: 404
			// TODO: print key array to w
		default:
			// TODO: 404
		}
	})
	return mux
}

func main() {
	mux := Mux()
	http.ListenAndServe(":9080", mux)
}
