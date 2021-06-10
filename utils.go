package main

import (
	"io/ioutil"
	"gopkg.in/yaml.v3"
)

func DetectSopsYaml(filename string) (bool, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return false, err
	}

	var sf map[string]interface{}
	err = yaml.Unmarshal(data, &sf)
	if err != nil {
	    return false, err
	}

	if _, ok := sf["sops"]; ok {
		return true, nil
	}
	return false, nil
}
