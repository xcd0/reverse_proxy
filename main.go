package main

import (
	"log"
)

func init() {
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

	config, err := parseArgs()
	if err != nil {
		log.Fatalf("Error parsing arguments: %v", err)
	}
	log.Printf("log : %v", config)

	generateServer(config)
}
