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
			"event":"delete"
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
	}{
		{
			name:            "get key which exists",
			url:             "/api/key1",
			method:          http.MethodGet,
			expResponseBody: "value1",
			expResponseCode: http.StatusOK,
		},
		{
			name:            "get key which has been deleted",
			url:             "/api/key2",
			method:          http.MethodGet,
			expResponseCode: http.StatusNoContent,
		},
		{
			name:            "get key which has never existed",
			url:             "/api/key3",
			method:          http.MethodGet,
			expResponseCode: http.StatusNotFound,
		},
		{
			name:   "get history for key which exists",
			url:    "/api/key1/history",
			method: http.MethodGet,
			expResponseBody: `[
	{
		"event":"create",
		"value":"value1"
	}
]
`,
			expResponseCode: http.StatusOK,
		},
		{
			name:            "get history for key which has never existed",
			url:             "/api/key1/history",
			method:          http.MethodGet,
			expResponseCode: http.StatusNotFound,
		},
		{
			name:            "post key which has never existed",
			url:             "/api",
			method:          http.MethodPost,
			reqBody:         `{"key3":"value3"}`,
			expResponseCode: http.StatusCreated,
		},
		{
			name:            "post key which already exists",
			url:             "/api",
			method:          http.MethodPost,
			reqBody:         `{"key1":"value1"}`,
			expResponseCode: http.StatusBadRequest,
		},
		{
			name:            "post key which has been deleted",
			url:             "/api",
			method:          http.MethodPost,
			reqBody:         `{"key2":"value2"}`,
			expResponseCode: http.StatusCreated,
		},
		{
			name:            "put update to key which exists",
			url:             "/api/key1",
			method:          http.MethodPut,
			reqBody:         "value4",
			expResponseCode: http.StatusNoContent,
		},
		{
			name:            "put update to key which has been deleted",
			url:             "/api/key2",
			method:          http.MethodPut,
			reqBody:         "value4",
			expResponseCode: http.StatusBadRequest,
		},
		{
			name:            "put update to key which has never existed",
			url:             "/api/key3",
			method:          http.MethodPut,
			reqBody:         "value4",
			expResponseCode: http.StatusNotFound,
		},
		{
			name:            "delete key which exists",
			url:             "/api/key1",
			method:          http.MethodDelete,
			expResponseCode: http.StatusNoContent,
		},
		{
			name:            "delete key which has already been deleted",
			url:             "/api/key2",
			method:          http.MethodDelete,
			expResponseCode: http.StatusBadRequest,
		},
		{
			name:            "delete key which has never existed",
			url:             "/api/key1",
			method:          http.MethodDelete,
			expResponseCode: http.StatusNotFound,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(dataFilePath, []byte(initialState), 0666); err != nil {
				assert.Fail(t, err.Error())
			}

			var reqBody io.Reader
			if tc.reqBody != "" {
				reqBody = bytes.NewReader([]byte(tc.reqBody))
			}
			req, err := http.NewRequest(tc.method, tc.url, reqBody)
			if err != nil {
				assert.Fail(t, err.Error())
			}

			resp := httptest.NewRecorder()
			Mux().ServeHTTP(resp, req)

			assert.Equal(t, tc.expResponseCode, resp.Code)
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				assert.Fail(t, err.Error())
			}
			assert.Equal(t, tc.expResponseBody, string(respBody))
		})
	}
}
