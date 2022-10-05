package main

import (
	"its-dave/simple-crud-rest-server/repository"
	"its-dave/simple-crud-rest-server/server"
	"net/http"
)

func main() {
	repo := repository.Repo{}
	repo.SetDataFilePath("data.json")
	http.ListenAndServe(":9080", server.Create(repo))
}
