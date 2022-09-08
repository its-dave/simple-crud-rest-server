package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	dataFilePath = "data.json"
)

type eventObj struct {
	Event string `json:"event"`
	Value string `json:"value"`
}

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
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			respBody, respCode := handleCreateReq(r)
			w.WriteHeader(respCode)
			fmt.Fprint(w, respBody)
			return
		case 2:
			// TODO: /api/key is called, must not be POST
			key := urlParts[1]
			switch r.Method {
			case http.MethodGet:
				// Get value for key

				respBody, respCode := handleReadReq(r, key)
				w.WriteHeader(respCode)
				fmt.Fprint(w, respBody)
				return
			case http.MethodPatch, http.MethodPut:
				// Update key:value

				respBody, respCode := handleUpdateReq(r, key)
				w.WriteHeader(respCode)
				fmt.Fprint(w, respBody)
				return
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

// handleUpdateReq handles a put/patch request and returns the desired response body and code
func handleUpdateReq(r *http.Request, key string) (string, int) {
	// Parse request body
	body := body(r)
	value := string(body)

	dataMap, err := storedData()
	if err != nil {
		return err.Error(), http.StatusInternalServerError
	}
	keyArray, exists := dataMap[key]
	if !exists {
		// Key does not exist
		return "", http.StatusNotFound
	}

	// Get value into a usable form
	array, ok := keyArray.([]interface{})
	if !ok {
		return err.Error(), http.StatusInternalServerError
	}
	latest := array[len(array)-1]
	latestJson, _ := json.Marshal(latest)
	var latestEventObj eventObj
	err = json.Unmarshal(latestJson, &latestEventObj)
	if err != nil {
		return err.Error(), http.StatusInternalServerError
	}
	if latestEventObj.Value == "" {
		return "TODO: key has been deleted", http.StatusBadRequest
	}

	// Set new key:value
	dataMap[key] = append(array, eventObj{
		Event: "update",
		Value: value,
	})

	err = writeData(dataMap)
	if err != nil {
		return err.Error(), http.StatusInternalServerError
	}
	return "", http.StatusNoContent
}

// handleCreateReq handles a post request and returns the desired response body and code
func handleCreateReq(r *http.Request) (string, int) {
	// Parse request body
	body := body(r)
	var bodyJson interface{}
	err := json.Unmarshal(body, &bodyJson)
	if err != nil {
		return "TODO: invalid post body", http.StatusBadRequest
	}
	bodyMap := bodyJson.(map[string]interface{})
	if len(bodyMap) != 1 {
		return "TODO: invalid post body", http.StatusBadRequest
	}

	dataMap, err := storedData()
	if err != nil {
		return err.Error(), http.StatusInternalServerError
	}

	// Set new key:value
	for key, valueInterface := range bodyMap {
		value, ok := valueInterface.(string)
		if !ok {
			return "TODO: invalid post body", http.StatusBadRequest
		}

		if _, exists := dataMap[key]; exists {
			// TODO: if key exists but has no value then append to array
			return "TODO: key already exists", http.StatusBadRequest
		}
		dataMap[key] = []eventObj{
			{
				Event: "create",
				Value: value,
			},
		}
	}

	err = writeData(dataMap)
	if err != nil {
		return err.Error(), http.StatusInternalServerError
	}
	return "", http.StatusAccepted
}

// handleReadReq handles a get request and returns the desired response body and code
func handleReadReq(r *http.Request, key string) (string, int) {
	dataMap, err := storedData()
	if err != nil {
		return err.Error(), http.StatusInternalServerError
	}
	keyArray, exists := dataMap[key]
	if !exists {
		// Key does not exist
		return "", http.StatusNotFound
	}

	// Get value into a usable form
	array, ok := keyArray.([]interface{})
	if !ok {
		return err.Error(), http.StatusInternalServerError
	}
	latest := array[len(array)-1]
	latestJson, _ := json.Marshal(latest)
	var latestEventObj eventObj
	err = json.Unmarshal(latestJson, &latestEventObj)
	if err != nil {
		return err.Error(), http.StatusInternalServerError
	}

	// Key has been deleted
	if latestEventObj.Value == "" {
		return "", http.StatusNoContent
	}

	return latestEventObj.Value, http.StatusOK
}

// storedData parses the stored JSON data and returns it as a map
func storedData() (map[string]interface{}, error) {
	// Parse stored data
	data, err := os.ReadFile(dataFilePath)
	if err != nil {
		return nil, err
	}
	var jsonData interface{}
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		return nil, err
	}
	return jsonData.(map[string]interface{}), nil
}

// writeData saves the specified JSON data to the data file
func writeData(dataMap map[string]interface{}) error {
	dataToWrite, err := json.Marshal(dataMap)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dataFilePath, dataToWrite, 0666); err != nil {
		return err
	}
	return nil
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
