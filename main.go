package main

import (
	"net/http"
)

func Mux() *http.ServeMux {
	mux := http.NewServeMux()
	return mux
}

func main() {
	mux := Mux()
	http.ListenAndServe(":9080", mux)
}
