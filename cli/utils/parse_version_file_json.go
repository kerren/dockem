package utils

import "encoding/json"

type Version struct {
	Version string `json:"version"`
}

func ParseVersionFileJson(jsonData []byte) (*Version, error) {
	var version Version
	err := json.Unmarshal(jsonData, &version)
	if err != nil {
		return nil, err
	}
	return &version, nil
}
