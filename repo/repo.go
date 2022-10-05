package repo

import (
	"encoding/json"
	"errors"
	"os"
)

const dataFilePath = "data.json"

// ReadData parses the stored JSON data and returns it as a map
func ReadData() (map[string]interface{}, error) {
	// Parse stored data
	data, err := os.ReadFile(dataFilePath)
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
	if err := WriteToDataFile(dataToWrite); err != nil {
		return err
	}
	return nil
}

// InitialiseData ensures that the data file exists, creating an empty JSON object if not
func InitialiseData() error {
	_, err := os.Stat(dataFilePath)
	if errors.Is(err, os.ErrNotExist) {
		WriteToDataFile([]byte("{}"))
	}
	return err
}

func WriteToDataFile(bytes []byte) error {
	return os.WriteFile(dataFilePath, bytes, 0666)
}
