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

	ErrorUnexpected      = "Unexpected error:"
	ErrorKeyDeleted      = "Error: the specified key has been deleted"
	ErrorKeyExists       = "Error: the specified key already exists"
	ErrorInvalidPutBody  = "Error: request body must be a single value"
	ErrorInvalidPostBody = "Error: request body must be of the form {\"key\":\"value\"}"
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
			switch r.Method {
			case http.MethodGet:
				// Get value for key

				respBody, respCode := handleReadReq(r, urlParts[1])
				w.WriteHeader(respCode)
				fmt.Fprint(w, respBody)
				return
			case http.MethodPatch, http.MethodPut:
				// Update key:value

				respBody, respCode := handleUpdateReq(r, urlParts[1])
				w.WriteHeader(respCode)
				fmt.Fprint(w, respBody)
				return
			case http.MethodDelete:
				// Delete value for key

				respBody, respCode := handleDeleteReq(r, urlParts[1])
				w.WriteHeader(respCode)
				fmt.Fprint(w, respBody)
				return
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
		case 3:
			// Get history for key

			if urlParts[2] != "history" {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			respBody, respCode := handleHistoryReq(r, urlParts[1])
			w.WriteHeader(respCode)
			fmt.Fprint(w, respBody)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	})
	return mux
}

// handleDeleteReq handles a delete request and returns the desired response body and code
func handleDeleteReq(r *http.Request, key string) (string, int) {
	dataMap, err := storedData()
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	keyArray, exists := dataMap[key]
	if !exists {
		// Key does not exist
		return "", http.StatusNotFound
	}

	// Get value into a usable form
	array, ok := keyArray.([]interface{})
	if !ok {
		return fmt.Sprint(ErrorUnexpected, "type assertion failed"), http.StatusInternalServerError
	}
	latest := array[len(array)-1]
	latestJson, _ := json.Marshal(latest)
	var latestEventObj eventObj
	err = json.Unmarshal(latestJson, &latestEventObj)
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	if latestEventObj.Value == "" {
		return ErrorKeyDeleted, http.StatusBadRequest
	}

	// Set new key:value
	dataMap[key] = append(array, eventObj{
		Event: "delete",
	})

	err = writeData(dataMap)
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	return "", http.StatusNoContent
}

// handleUpdateReq handles a put/patch request and returns the desired response body and code
func handleUpdateReq(r *http.Request, key string) (string, int) {
	// Parse request body
	body, err := body(r)
	if body == nil {
		return ErrorInvalidPutBody, http.StatusBadRequest
	}
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	value := string(body)

	dataMap, err := storedData()
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	keyArray, exists := dataMap[key]
	if !exists {
		// Key does not exist
		return "", http.StatusNotFound
	}

	// Get value into a usable form
	array, ok := keyArray.([]interface{})
	if !ok {
		return fmt.Sprint(ErrorUnexpected, "type assertion failed"), http.StatusInternalServerError
	}
	latest := array[len(array)-1]
	latestJson, _ := json.Marshal(latest)
	var latestEventObj eventObj
	err = json.Unmarshal(latestJson, &latestEventObj)
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	if latestEventObj.Value == "" {
		return ErrorKeyDeleted, http.StatusBadRequest
	}

	// Set new key:value
	dataMap[key] = append(array, eventObj{
		Event: "update",
		Value: value,
	})

	err = writeData(dataMap)
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	return "", http.StatusNoContent
}

// handleCreateReq handles a post request and returns the desired response body and code
func handleCreateReq(r *http.Request) (string, int) {
	// Parse request body
	body, err := body(r)
	if body == nil {
		return ErrorInvalidPostBody, http.StatusBadRequest
	}
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	var bodyJson interface{}
	if err = json.Unmarshal(body, &bodyJson); err != nil {
		return ErrorInvalidPostBody, http.StatusBadRequest
	}
	bodyMap := bodyJson.(map[string]interface{})
	if len(bodyMap) != 1 {
		return ErrorInvalidPostBody, http.StatusBadRequest
	}

	dataMap, err := storedData()
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}

	for key, valueInterface := range bodyMap {
		value, ok := valueInterface.(string)
		if !ok {
			return ErrorInvalidPostBody, http.StatusBadRequest
		}
		event := eventObj{
			Event: "create",
			Value: value,
		}

		keyArray, exists := dataMap[key]
		if !exists {
			// Set new key:value
			dataMap[key] = []eventObj{event}
			continue
		}

		// Get value into a usable form
		array, ok := keyArray.([]interface{})
		if !ok {
			return fmt.Sprint(ErrorUnexpected, "type assertion failed"), http.StatusInternalServerError
		}
		latest := array[len(array)-1]
		latestJson, _ := json.Marshal(latest)
		var latestEventObj eventObj
		err = json.Unmarshal(latestJson, &latestEventObj)
		if err != nil {
			return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
		}

		if latestEventObj.Value != "" {
			return ErrorKeyExists, http.StatusBadRequest
		}

		// Set new key:value
		dataMap[key] = append(array, event)
	}

	err = writeData(dataMap)
	if err != nil {
		return err.Error(), http.StatusInternalServerError
	}
	return "", http.StatusCreated
}

// handleReadReq handles a get request and returns the desired response body and code
func handleReadReq(r *http.Request, key string) (string, int) {
	dataMap, err := storedData()
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	keyArray, exists := dataMap[key]
	if !exists {
		// Key does not exist
		return "", http.StatusNotFound
	}

	// Get value into a usable form
	array, ok := keyArray.([]interface{})
	if !ok {
		return fmt.Sprint(ErrorUnexpected, "type assertion failed"), http.StatusInternalServerError
	}
	latest := array[len(array)-1]
	latestJson, _ := json.Marshal(latest)
	var latestEventObj eventObj
	err = json.Unmarshal(latestJson, &latestEventObj)
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}

	// Key has been deleted
	if latestEventObj.Value == "" {
		return "", http.StatusNoContent
	}

	return latestEventObj.Value, http.StatusOK
}

// handleHistoryReq handles a get history request and returns the desired response body and code
func handleHistoryReq(r *http.Request, key string) (string, int) {
	dataMap, err := storedData()
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	keyArray, exists := dataMap[key]
	if !exists {
		// Key does not exist
		return "", http.StatusNotFound
	}

	array, err := json.Marshal(keyArray)
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}

	return string(array), http.StatusOK
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

// body gets the body data from the specified request
func body(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return nil, err
	}
	return body, nil
}

func main() {
	mux := Mux()
	http.ListenAndServe(":9080", mux)
}
