package main

import (
	"encoding/json"
	"io/ioutil"
)

var items map[string]int = make(map[string]int)

func parseItems(file string) error {
	f, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	temp := make(map[string]int, 400)

	err = json.Unmarshal(f, &temp)
	if err != nil {
		return err
	}

	items = temp

	return nil
}
