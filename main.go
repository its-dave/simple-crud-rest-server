package main

import (
	"encoding/json"
	"errors"
	"io"
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
			// Create new key:value

			if r.Method != http.MethodPost {
				// TODO: 405
				return
			}

			// Parse request body
			body := body(r)
			var jsonBody interface{}
			err = json.Unmarshal(body, &jsonBody)
			if err != nil {
				// TODO: 400
				return
			}
			mapBody := jsonBody.(map[string]interface{})
			if len(mapBody) != 1 {
				// TODO: 400
				return
			}

			// Parse stored data
			data, err := os.ReadFile(dataFilePath)
			if err != nil {
				// TODO: 500
				return
			}
			var jsonData interface{}
			err = json.Unmarshal(data, &jsonData)
			if err != nil {
				// TODO: 500
				return
			}
			mapData := jsonData.(map[string]interface{})

			// Set new key:value
			for key, valueInterface := range mapBody {
				value, ok := valueInterface.(string)
				if !ok {
					// TODO: 400
					return
				}

				if _, exists := mapData[key]; exists {
					// TODO: if key exists but has no value then append to array
					// TODO: 400
					return
				}
				mapData[key] = []struct {
					Event string `json:"event"`
					Value string `json:"value"`
				}{
					{
						Event: "create",
						Value: value,
					},
				}
			}

			// Write data
			dataToWrite, err := json.Marshal(mapData)
			if err != nil {
				// TODO: 500
				return
			}
			if err := os.WriteFile(dataFilePath, dataToWrite, 0666); err != nil {
				// TODO: 500
				return
			}

			// TODO: 202
			return
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

func body(r *http.Request) []byte {
	if r.Body == nil {
		// TODO: 400
		return nil
	}
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		// TODO: 500
		panic(err)
	}
	return body
}

func main() {
	mux := Mux()
	http.ListenAndServe(":9080", mux)
}
