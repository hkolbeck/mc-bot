package main

import (
	"fmt"
	)

func main() {
	conf, err := ReadConfig("/home/cbeck/mc-bot/nix_example.json")
	if err == nil {
		fmt.Printf("%#v\n", conf)
	} else {
		fmt.Println(err)
	}
}