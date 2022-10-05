package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"its-dave/simple-crud-rest-server/repo"
	"net/http"
	"strings"
)

const (
	contentType     = "Content-Type"
	contentTypeText = "text/plain"
	contentTypeJson = "application/json"

	ErrorUnexpected      = "Unexpected error:"
	ErrorKeyDeleted      = "Error: the specified key has been deleted"
	ErrorKeyExists       = "Error: the specified key already exists"
	ErrorInvalidPutBody  = "Error: request body must be a single value with Content-Type text/plain"
	ErrorInvalidPostBody = "Error: request body must be of the form {\"key\":\"value\"} with Content-Type application/json"
)

type eventObj struct {
	Event string `json:"event"`
	Value string `json:"value"`
}

// Mux creates a simple rest server using an existing data file if found
func Mux() *http.ServeMux {
	if err := repo.InitialiseData(); err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api", handlePostFunc)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		url := strings.TrimPrefix(r.URL.Path, "/")
		url = strings.TrimSuffix(url, "/")
		urlParts := strings.Split(url, "/")
		switch len(urlParts) {
		case 1:
			handlePostFunc(w, r)
			return
		case 2:
			switch r.Method {
			case http.MethodGet:
				// Get value for key

				respBody, respCode := handleReadReq(r, urlParts[1])
				w.Header().Add(contentType, contentTypeText)
				w.WriteHeader(respCode)
				fmt.Fprint(w, respBody)
				return
			case http.MethodPatch, http.MethodPut:
				// Update key:value

				respBody, respCode := handleUpdateReq(r, urlParts[1])
				w.Header().Add(contentType, contentTypeText)
				w.WriteHeader(respCode)
				fmt.Fprint(w, respBody)
				return
			case http.MethodDelete:
				// Delete value for key

				respBody, respCode := handleDeleteReq(r, urlParts[1])
				w.Header().Add(contentType, contentTypeText)
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
			w.Header().Add(contentType, contentTypeJson)
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

// handlePostFunc handles post requests
func handlePostFunc(w http.ResponseWriter, r *http.Request) {
	// Create new key:value

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	respBody, respCode := handleCreateReq(r)
	w.Header().Add(contentType, contentTypeText)
	w.WriteHeader(respCode)
	fmt.Fprint(w, respBody)
}

// handleDeleteReq handles a delete request and returns the desired response body and code
func handleDeleteReq(r *http.Request, key string) (string, int) {
	dataMap, err := repo.ReadData()
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	keyArray, exists := dataMap[key]
	if !exists {
		// Key does not exist
		return "", http.StatusNotFound
	}

	array, err := sliceFromArray(keyArray)
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	latestEventObj, err := latestEventFromSlice(array)
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

	err = repo.WriteData(dataMap)
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	return "", http.StatusNoContent
}

// handleUpdateReq handles a put/patch request and returns the desired response body and code
func handleUpdateReq(r *http.Request, key string) (string, int) {
	if r.Header.Get(contentType) != contentTypeText {
		return ErrorInvalidPutBody, http.StatusUnsupportedMediaType
	}

	// Parse request body
	body, err := body(r)
	if body == nil {
		return ErrorInvalidPutBody, http.StatusBadRequest
	}
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	value := string(body)

	dataMap, err := repo.ReadData()
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	keyArray, exists := dataMap[key]
	if !exists {
		// Key does not exist
		return "", http.StatusNotFound
	}

	array, err := sliceFromArray(keyArray)
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	latestEventObj, err := latestEventFromSlice(array)
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

	err = repo.WriteData(dataMap)
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	return "", http.StatusNoContent
}

// handleCreateReq handles a post request and returns the desired response body and code
func handleCreateReq(r *http.Request) (string, int) {
	if r.Header.Get(contentType) != contentTypeJson {
		return ErrorInvalidPostBody, http.StatusUnsupportedMediaType
	}

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

	dataMap, err := repo.ReadData()
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

		array, err := sliceFromArray(keyArray)
		if err != nil {
			return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
		}
		latestEventObj, err := latestEventFromSlice(array)
		if err != nil {
			return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
		}

		if latestEventObj.Value != "" {
			return ErrorKeyExists, http.StatusBadRequest
		}

		// Set new key:value
		dataMap[key] = append(array, event)
	}

	err = repo.WriteData(dataMap)
	if err != nil {
		return err.Error(), http.StatusInternalServerError
	}
	return "", http.StatusCreated
}

// handleReadReq handles a get request and returns the desired response body and code
func handleReadReq(r *http.Request, key string) (string, int) {
	dataMap, err := repo.ReadData()
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	keyArray, exists := dataMap[key]
	if !exists {
		// Key does not exist
		return "", http.StatusNotFound
	}

	array, err := sliceFromArray(keyArray)
	if err != nil {
		return fmt.Sprint(ErrorUnexpected, err.Error()), http.StatusInternalServerError
	}
	latestEventObj, err := latestEventFromSlice(array)
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
	dataMap, err := repo.ReadData()
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

// sliceFromArray parses the specified data for a specific key and returns it as a slice
func sliceFromArray(keyArray interface{}) ([]interface{}, error) {
	array, ok := keyArray.([]interface{})
	if !ok {
		return nil, errors.New("type assertion failed")
	}
	return array, nil
}

// latestEventFromSlice parses the specified slice for a key and returns the final element
func latestEventFromSlice(array []interface{}) (eventObj, error) {
	var latestEventObj eventObj
	latest := array[len(array)-1]
	latestJson, _ := json.Marshal(latest)
	err := json.Unmarshal(latestJson, &latestEventObj)
	if err != nil {
		return latestEventObj, err
	}
	return latestEventObj, nil
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
