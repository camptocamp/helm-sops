package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

func ReadAndUnmarshalYaml(filename string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var sf map[string]interface{}
	err = yaml.Unmarshal(data, &sf)
	if err != nil {
		return nil, err
	}

	return sf, nil
}

func DetectSopsKey(sf map[string]interface{}) bool {
	_, ok := sf["sops"]
	return ok
}
