package utils

import "encoding/json"

type Version struct {
	Version string `json:"version"`
}

// ParseVersionFileJson Parses a JSON byte slice into a `Version` struct.
func ParseVersionFileJson(jsonData []byte) (*Version, error) {
	var version Version
	err := json.Unmarshal(jsonData, &version)
	if err != nil {
		return nil, err
	}
	return &version, nil
}
