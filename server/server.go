package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"its-dave/simple-crud-rest-server/repository"
	"net/http"
	"strings"
)

const (
	contentType     = "Content-Type"
	contentTypeText = "text/plain"
	contentTypeJson = "application/json"

	errorUnexpected      = "Unexpected error:"
	errorKeyDeleted      = "Error: the specified key has been deleted"
	errorKeyExists       = "Error: the specified key already exists"
	errorInvalidPutBody  = "Error: request body must be a single value with Content-Type text/plain"
	errorInvalidPostBody = "Error: request body must be of the form {\"key\":\"value\"} with Content-Type application/json"
)

type eventObj struct {
	Event string `json:"event"`
	Value string `json:"value"`
}

// Create returns a simple rest server mux using an existing data file if found
func Create(repo repository.Repo) *http.ServeMux {
	if err := repo.InitialiseData(); err != nil {
		panic(err)
	}

	handlePostFunc := func(w http.ResponseWriter, r *http.Request) {
		// Create new key:value

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		respBody, respCode := handleCreateReq(repo, r)
		w.Header().Add(contentType, contentTypeText)
		w.WriteHeader(respCode)
		fmt.Fprint(w, respBody)
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

				respBody, respCode := handleReadReq(repo, r, urlParts[1])
				w.Header().Add(contentType, contentTypeText)
				w.WriteHeader(respCode)
				fmt.Fprint(w, respBody)
				return
			case http.MethodPatch, http.MethodPut:
				// Update key:value

				respBody, respCode := handleUpdateReq(repo, r, urlParts[1])
				w.Header().Add(contentType, contentTypeText)
				w.WriteHeader(respCode)
				fmt.Fprint(w, respBody)
				return
			case http.MethodDelete:
				// Delete value for key

				respBody, respCode := handleDeleteReq(repo, r, urlParts[1])
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

			respBody, respCode := handleHistoryReq(repo, r, urlParts[1])
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

// handleDeleteReq handles a delete request and returns the desired response body and code
func handleDeleteReq(repo repository.Repo, r *http.Request, key string) (string, int) {
	dataMap, err := repo.ReadData()
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
	}
	keyArray, exists := dataMap[key]
	if !exists {
		// Key does not exist
		return "", http.StatusNotFound
	}

	array, err := sliceFromArray(keyArray)
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
	}
	latestEventObj, err := latestEventFromSlice(array)
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
	}

	if latestEventObj.Value == "" {
		return errorKeyDeleted, http.StatusBadRequest
	}

	// Set new key:value
	dataMap[key] = append(array, eventObj{
		Event: "delete",
	})

	err = repo.WriteData(dataMap)
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
	}
	return "", http.StatusNoContent
}

// handleUpdateReq handles a put/patch request and returns the desired response body and code
func handleUpdateReq(repo repository.Repo, r *http.Request, key string) (string, int) {
	if r.Header.Get(contentType) != contentTypeText {
		return errorInvalidPutBody, http.StatusUnsupportedMediaType
	}

	// Parse request body
	body, err := body(r)
	if body == nil {
		return errorInvalidPutBody, http.StatusBadRequest
	}
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
	}
	value := string(body)

	dataMap, err := repo.ReadData()
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
	}
	keyArray, exists := dataMap[key]
	if !exists {
		// Key does not exist
		return "", http.StatusNotFound
	}

	array, err := sliceFromArray(keyArray)
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
	}
	latestEventObj, err := latestEventFromSlice(array)
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
	}

	if latestEventObj.Value == "" {
		return errorKeyDeleted, http.StatusBadRequest
	}

	// Set new key:value
	dataMap[key] = append(array, eventObj{
		Event: "update",
		Value: value,
	})

	err = repo.WriteData(dataMap)
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
	}
	return "", http.StatusNoContent
}

// handleCreateReq handles a post request and returns the desired response body and code
func handleCreateReq(repo repository.Repo, r *http.Request) (string, int) {
	if r.Header.Get(contentType) != contentTypeJson {
		return errorInvalidPostBody, http.StatusUnsupportedMediaType
	}

	// Parse request body
	body, err := body(r)
	if body == nil {
		return errorInvalidPostBody, http.StatusBadRequest
	}
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
	}
	var bodyJson interface{}
	if err = json.Unmarshal(body, &bodyJson); err != nil {
		return errorInvalidPostBody, http.StatusBadRequest
	}
	bodyMap := bodyJson.(map[string]interface{})
	if len(bodyMap) != 1 {
		return errorInvalidPostBody, http.StatusBadRequest
	}

	dataMap, err := repo.ReadData()
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
	}

	for key, valueInterface := range bodyMap {
		value, ok := valueInterface.(string)
		if !ok {
			return errorInvalidPostBody, http.StatusBadRequest
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
			return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
		}
		latestEventObj, err := latestEventFromSlice(array)
		if err != nil {
			return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
		}

		if latestEventObj.Value != "" {
			return errorKeyExists, http.StatusBadRequest
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
func handleReadReq(repo repository.Repo, r *http.Request, key string) (string, int) {
	dataMap, err := repo.ReadData()
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
	}
	keyArray, exists := dataMap[key]
	if !exists {
		// Key does not exist
		return "", http.StatusNotFound
	}

	array, err := sliceFromArray(keyArray)
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
	}
	latestEventObj, err := latestEventFromSlice(array)
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
	}

	// Key has been deleted
	if latestEventObj.Value == "" {
		return "", http.StatusNoContent
	}

	return latestEventObj.Value, http.StatusOK
}

// handleHistoryReq handles a get history request and returns the desired response body and code
func handleHistoryReq(repo repository.Repo, r *http.Request, key string) (string, int) {
	dataMap, err := repo.ReadData()
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
	}
	keyArray, exists := dataMap[key]
	if !exists {
		// Key does not exist
		return "", http.StatusNotFound
	}

	array, err := json.Marshal(keyArray)
	if err != nil {
		return fmt.Sprint(errorUnexpected, err.Error()), http.StatusInternalServerError
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
