package repo

import (
	"encoding/json"
	"errors"
	"os"
)

const DataFilePath = "data.json"

// ReadData parses the stored JSON data and returns it as a map
func ReadData() (map[string]interface{}, error) {
	// Parse stored data
	data, err := os.ReadFile(DataFilePath)
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
func WriteData(dataMap map[string]interface{}) error {
	dataToWrite, err := json.Marshal(dataMap)
	if err != nil {
		return err
	}
	if err := os.WriteFile(DataFilePath, dataToWrite, 0666); err != nil {
		return err
	}
	return nil
}

// InitialiseData ensures that the data file exists
func InitialiseData() error {
	_, err := os.Stat(DataFilePath)
	if errors.Is(err, os.ErrNotExist) {
		return os.WriteFile(DataFilePath, []byte("{}"), 0666)
	}
	return err
}
