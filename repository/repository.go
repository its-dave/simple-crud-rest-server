package repository

import (
	"encoding/json"
	"errors"
	"os"
)

type Repo struct {
	dataFilePath string
}

func (repo *Repo) SetDataFilePath(dataFilePath string) {
	repo.dataFilePath = dataFilePath
}

// ReadData parses the stored JSON data and returns it as a map
func (repo Repo) ReadData() (map[string]interface{}, error) {
	// Parse stored data
	data, err := os.ReadFile(repo.dataFilePath)
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

// WriteData saves the specified JSON data to the data file
func (repo Repo) WriteData(dataMap map[string]interface{}) error {
	dataToWrite, err := json.Marshal(dataMap)
	if err != nil {
		return err
	}
	if err := repo.WriteToDataFile(dataToWrite); err != nil {
		return err
	}
	return nil
}

// InitialiseData ensures that the data file exists, creating an empty JSON object if not
func (repo Repo) InitialiseData() error {
	_, err := os.Stat(repo.dataFilePath)
	if errors.Is(err, os.ErrNotExist) {
		repo.WriteToDataFile([]byte("{}"))
	}
	return err
}

func (repo Repo) WriteToDataFile(bytes []byte) error {
	return os.WriteFile(repo.dataFilePath, bytes, 0666)
}
