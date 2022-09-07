package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	for _, tc := range []struct {
		name            string
		url             string
		method          string
		expResponseCode int
		jsonBody        string
	}{
		{
			name:            "get",
			url:             "/api/asdf",
			method:          http.MethodGet,
			expResponseCode: http.StatusOK,
		},
		{
			name:            "get history",
			url:             "/api/asdf/history",
			method:          http.MethodGet,
			expResponseCode: http.StatusOK,
		},
		{
			name:            "post",
			url:             "/api",
			method:          http.MethodPost,
			expResponseCode: http.StatusOK,
			jsonBody:        `{"test":"example"}`,
		},
		{
			name:            "put",
			url:             "/api/asdf",
			method:          http.MethodPut,
			expResponseCode: http.StatusOK,
			jsonBody:        `{"test":"example"}`,
		},
		{
			name:            "delete",
			url:             "/api/asdf",
			method:          http.MethodDelete,
			expResponseCode: http.StatusOK,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body io.Reader
			if tc.jsonBody != "" {
				body = bytes.NewReader([]byte(tc.jsonBody))
			}
			req, err := http.NewRequest(tc.method, tc.url, body)
			if err != nil {
				assert.Fail(t, err.Error())
			}

			resp := httptest.NewRecorder()
			Mux().ServeHTTP(resp, req)

			assert.Equal(t, tc.expResponseCode, resp.Code)
		})
	}
}
