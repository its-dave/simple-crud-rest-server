package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	initialState := `{
	"key1":[
		{
			"event":"create",
			"value":"value1"
		}
	],
	"key2":[
		{
			"event":"create",
			"value":"value1"
		},
		{
			"event":"delete",
			"value":""
		}
	]
}
`
	for _, tc := range []struct {
		name            string
		url             string
		method          string
		reqBody         string
		expResponseBody string
		expResponseCode int
		expContentType  string
	}{
		{
			name:            "get key which exists",
			url:             "/api/key1",
			method:          http.MethodGet,
			expResponseBody: "value1",
			expResponseCode: http.StatusOK,
			expContentType:  contentTypeText,
		},
		{
			name:            "get key which has been deleted",
			url:             "/api/key2",
			method:          http.MethodGet,
			expResponseCode: http.StatusNoContent,
			expContentType:  contentTypeText,
		},
		{
			name:            "get key which has never existed",
			url:             "/api/key3",
			method:          http.MethodGet,
			expResponseCode: http.StatusNotFound,
			expContentType:  contentTypeText,
		},
		{
			name:            "get wrong endpoint",
			url:             "/api/",
			method:          http.MethodGet,
			expResponseCode: http.StatusMethodNotAllowed,
		},
		{
			name:            "get history for key which exists",
			url:             "/api/key1/history",
			method:          http.MethodGet,
			expResponseBody: `[{"event":"create","value":"value1"}]`,
			expResponseCode: http.StatusOK,
			expContentType:  contentTypeJson,
		},
		{
			name:            "get history for key which has never existed",
			url:             "/api/key3/history",
			method:          http.MethodGet,
			expResponseCode: http.StatusNotFound,
			expContentType:  contentTypeJson,
		},
		{
			name:            "get too long endpoint",
			url:             "/api/key1/history/other",
			method:          http.MethodGet,
			expResponseCode: http.StatusNotFound,
		},
		{
			name:            "get invalid endpoint",
			url:             "/api/key1/incorrect",
			method:          http.MethodGet,
			expResponseCode: http.StatusNotFound,
		},
		{
			name:            "post key which has never existed",
			url:             "/api/",
			method:          http.MethodPost,
			reqBody:         `{"key3":"value3"}`,
			expResponseCode: http.StatusCreated,
			expContentType:  contentTypeText,
		},
		{
			name:            "post key which has never existed (no trailing /)",
			url:             "/api",
			method:          http.MethodPost,
			reqBody:         `{"key3":"value3"}`,
			expResponseCode: http.StatusCreated,
			expContentType:  contentTypeText,
		},
		{
			name:            "post key which already exists",
			url:             "/api/",
			method:          http.MethodPost,
			reqBody:         `{"key1":"value1"}`,
			expResponseCode: http.StatusBadRequest,
			expResponseBody: ErrorKeyExists,
			expContentType:  contentTypeText,
		},
		{
			name:            "post key which already exists (no trailing /)",
			url:             "/api",
			method:          http.MethodPost,
			reqBody:         `{"key1":"value1"}`,
			expResponseCode: http.StatusBadRequest,
			expResponseBody: ErrorKeyExists,
			expContentType:  contentTypeText,
		},
		{
			name:            "post key which has been deleted",
			url:             "/api/",
			method:          http.MethodPost,
			reqBody:         `{"key2":"value2"}`,
			expResponseCode: http.StatusCreated,
			expContentType:  contentTypeText,
		},
		{
			name:            "post key which has been deleted (no trailing /)",
			url:             "/api",
			method:          http.MethodPost,
			reqBody:         `{"key2":"value2"}`,
			expResponseCode: http.StatusCreated,
			expContentType:  contentTypeText,
		},
		{
			name:            "post to wrong endpoint",
			url:             "/api/key3",
			method:          http.MethodPost,
			reqBody:         `{"key3":"value3"}`,
			expResponseCode: http.StatusMethodNotAllowed,
		},
		{
			name:            "post with bad request body",
			url:             "/api/",
			method:          http.MethodPost,
			reqBody:         "key3",
			expResponseCode: http.StatusBadRequest,
			expResponseBody: ErrorInvalidPostBody,
			expContentType:  contentTypeText,
		},
		{
			name:            "put update to key which exists",
			url:             "/api/key1",
			method:          http.MethodPut,
			reqBody:         "value4",
			expResponseCode: http.StatusNoContent,
			expContentType:  contentTypeText,
		},
		{
			name:            "put update to key which has been deleted",
			url:             "/api/key2",
			method:          http.MethodPut,
			reqBody:         "value4",
			expResponseCode: http.StatusBadRequest,
			expResponseBody: ErrorKeyDeleted,
			expContentType:  contentTypeText,
		},
		{
			name:            "put update to key which has never existed",
			url:             "/api/key3",
			method:          http.MethodPut,
			reqBody:         "value4",
			expResponseCode: http.StatusNotFound,
			expContentType:  contentTypeText,
		},
		{
			name:            "put to wrong endpoint",
			url:             "/api/",
			method:          http.MethodPut,
			reqBody:         "value4",
			expResponseCode: http.StatusMethodNotAllowed,
		},
		{
			name:            "put with bad request body",
			url:             "/api/key1",
			method:          http.MethodPut,
			reqBody:         "",
			expResponseCode: http.StatusBadRequest,
			expResponseBody: ErrorInvalidPutBody,
			expContentType:  contentTypeText,
		},
		{
			name:            "delete key which exists",
			url:             "/api/key1",
			method:          http.MethodDelete,
			expResponseCode: http.StatusNoContent,
			expContentType:  contentTypeText,
		},
		{
			name:            "delete key which has already been deleted",
			url:             "/api/key2",
			method:          http.MethodDelete,
			expResponseCode: http.StatusBadRequest,
			expResponseBody: ErrorKeyDeleted,
			expContentType:  contentTypeText,
		},
		{
			name:            "delete key which has never existed",
			url:             "/api/key3",
			method:          http.MethodDelete,
			expResponseCode: http.StatusNotFound,
			expContentType:  contentTypeText,
		},
		{
			name:            "delete to wrong endpoint",
			url:             "/api/",
			method:          http.MethodDelete,
			reqBody:         "key1",
			expResponseCode: http.StatusMethodNotAllowed,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			initialiseData(t, initialState)
			requestAndCheckResponse(t, Mux(), tc.method, tc.url, tc.reqBody, tc.expResponseCode, tc.expResponseBody, tc.expContentType)
		})
	}
}

func Test_CRURDRH(t *testing.T) {
	initialiseData(t, "{}")
	mux := Mux()
	// Set key1:value1
	requestAndCheckResponse(t, mux, http.MethodPost, "/api/", `{"key1":"value1"}`, http.StatusCreated, "", contentTypeText)
	// Verify key1:value1
	requestAndCheckResponse(t, mux, http.MethodGet, "/api/key1", "", http.StatusOK, "value1", contentTypeText)
	// Set key1:value2
	requestAndCheckResponse(t, mux, http.MethodPut, "/api/key1", "value2", http.StatusNoContent, "", contentTypeText)
	// Verify key1:value2
	requestAndCheckResponse(t, mux, http.MethodGet, "/api/key1", "", http.StatusOK, "value2", contentTypeText)
	// Delete key1
	requestAndCheckResponse(t, mux, http.MethodDelete, "/api/key1", "", http.StatusNoContent, "", contentTypeText)
	// Verify key1 unset
	requestAndCheckResponse(t, mux, http.MethodGet, "/api/key1", "", http.StatusNoContent, "", contentTypeText)
	// Verify key1 history
	requestAndCheckResponse(t, mux, http.MethodGet, "/api/key1/history", "", http.StatusOK, `[{"event":"create","value":"value1"},{"event":"update","value":"value2"},{"event":"delete","value":""}]`, contentTypeJson)
}

func Test_CDCUH(t *testing.T) {
	initialiseData(t, "{}")
	mux := Mux()
	// Set key1:value1
	requestAndCheckResponse(t, mux, http.MethodPost, "/api/", `{"key1":"value1"}`, http.StatusCreated, "", contentTypeText)
	// Delete key1
	requestAndCheckResponse(t, mux, http.MethodDelete, "/api/key1", "", http.StatusNoContent, "", contentTypeText)
	// Set key1:value1
	requestAndCheckResponse(t, mux, http.MethodPost, "/api/", `{"key1":"value1"}`, http.StatusCreated, "", contentTypeText)
	// Set key1:value2
	requestAndCheckResponse(t, mux, http.MethodPut, "/api/key1", "value2", http.StatusNoContent, "", contentTypeText)
	// Verify key1 history
	requestAndCheckResponse(t, mux, http.MethodGet, "/api/key1/history", "", http.StatusOK, `[{"event":"create","value":"value1"},{"event":"delete","value":""},{"event":"create","value":"value1"},{"event":"update","value":"value2"}]`, contentTypeJson)
}

func Test_CRCRURDRHH(t *testing.T) {
	initialiseData(t, "{}")
	mux := Mux()
	// Set key1:value1
	requestAndCheckResponse(t, mux, http.MethodPost, "/api/", `{"key1":"value1"}`, http.StatusCreated, "", contentTypeText)
	// Verify key1:value1
	requestAndCheckResponse(t, mux, http.MethodGet, "/api/key1", "", http.StatusOK, "value1", contentTypeText)
	// Set key2:value3
	requestAndCheckResponse(t, mux, http.MethodPost, "/api/", `{"key2":"value3"}`, http.StatusCreated, "", contentTypeText)
	// Verify key2:value3
	requestAndCheckResponse(t, mux, http.MethodGet, "/api/key2", "", http.StatusOK, "value3", contentTypeText)
	// Set key1:value2
	requestAndCheckResponse(t, mux, http.MethodPut, "/api/key1", "value2", http.StatusNoContent, "", contentTypeText)
	// Verify key1:value2
	requestAndCheckResponse(t, mux, http.MethodGet, "/api/key1", "", http.StatusOK, "value2", contentTypeText)
	// Delete key2
	requestAndCheckResponse(t, mux, http.MethodDelete, "/api/key2", "", http.StatusNoContent, "", contentTypeText)
	// Verify key2 unset
	requestAndCheckResponse(t, mux, http.MethodGet, "/api/key2", "", http.StatusNoContent, "", contentTypeText)
	// Verify key1 history
	requestAndCheckResponse(t, mux, http.MethodGet, "/api/key1/history", "", http.StatusOK, `[{"event":"create","value":"value1"},{"event":"update","value":"value2"}]`, contentTypeJson)
	// Verify key2 history
	requestAndCheckResponse(t, mux, http.MethodGet, "/api/key2/history", "", http.StatusOK, `[{"event":"create","value":"value3"},{"event":"delete","value":""}]`, contentTypeJson)
}

// initialiseData sets the data file to the specified data to ensure a known testing state
func initialiseData(t *testing.T, data string) {
	if err := os.WriteFile(dataFilePath, []byte(data), 0666); err != nil {
		assert.Fail(t, err.Error())
	}
}

// requestAndCheckResponse makes the specified request to the specified mux and asserts the specified response code, body, and content type
func requestAndCheckResponse(t *testing.T, mux *http.ServeMux, reqMethod, reqUrl, reqBody string, expRespCode int, expRespBody string, expRespContentType string) {
	var reqBodyBytes io.Reader
	if reqBody != "" {
		reqBodyBytes = bytes.NewReader([]byte(reqBody))
	}
	req, err := http.NewRequest(reqMethod, reqUrl, reqBodyBytes)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	assert.Equal(t, expRespCode, resp.Code)
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		assert.Fail(t, err.Error())
	}
	assert.Equal(t, expRespBody, string(respBody))
	assert.Equal(t, expRespContentType, string(resp.Result().Header.Get("Content-Type")))
}
