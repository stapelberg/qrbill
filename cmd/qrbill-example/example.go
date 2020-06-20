package main

import (
	"log"

	"github.com/stapelberg/qrbill"
)

func logic() error {
	return qrbill.Generate()
	//return nil
}

func main() {
	if err := logic(); err != nil {
		log.Fatal(err)
	}
}
