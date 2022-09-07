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
			name:            "get",
			url:             "/api/key1",
			method:          http.MethodGet,
			expResponseBody: "value1",
			expResponseCode: http.StatusOK,
		},
		{
			name:   "get history",
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
			name:            "post",
			url:             "/api",
			method:          http.MethodPost,
			reqBody:         `{"key2":"value2"}`,
			expResponseCode: http.StatusOK,
		},
		{
			name:            "put",
			url:             "/api/key1",
			method:          http.MethodPut,
			reqBody:         "value2",
			expResponseCode: http.StatusOK,
		},
		{
			name:            "delete",
			url:             "/api/key1",
			method:          http.MethodDelete,
			expResponseCode: http.StatusOK,
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
