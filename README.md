# simple-crud-rest-server

## A basic REST API to Create, Read, Update and Delete key-value pairs
Written in Go using minimal third-party libraries

### Endpoints

- `POST /api {"key1":"value1"}` - create a new entry with key `key1` and value `value1`
- `PUT /api/key1 value2` - update existing entry with key `key1` to have value `value2`
- `GET /api/key1` - get the value of key `key1`
- `DELETE /api/key1` - delete the value associated with `key1`
- `GET /api/key1/history` - get a history of events which have been issued for key `key1`

### Running

- Clone this repo and run the server with `go run main.go`
    - The server will run on `localhost:9080/`
- Run the unit tests with `go test`
- Expected behaviour can be seen by reading the unit tests

### Nuances

- The server will save the stored data in a simple JSON file - this will be created if it does not already exist and persists when the server is stopped
- `DELETE` requests delete the value not the key - this can be seen by then getting the history
    - You must call `POST` to create a new value or `PUT` to update a non-deleted value
    - Technically you could use a `PUT` request to set a value to be an empty string which would functionally be the same as a `DELETE` request
- There is currently no support for different users, any request can affect any key
    - A basic authentication system could be added, giving each user a separate namespace which only they could access
- Currently only string values are supported
    - While the data is stored as JSON it would be trivial to expand to supporting JSON objects and arrays
    - More complex data structures could be added as documented structs
- As a simple project this server has a few potential bottlenecks
    - Reading and writing the whole file on each request will quickly become slow, adding a database to store the data would solve this
    - Running the server in a container in Kubernetes could allow for easy scaling and redundancy, and a gateway of some kind could provide rate limiting as well as authentication
