package main

import (
	"io"
	"log"
	"os"
)

func init() {
	log.SetFlags(log.Ltime | log.Lshortfile)
}

func main() {

	config, err := parseArgs()
	set_logfile(config)

	if err != nil {
		log.Fatalf("Error parsing arguments: %v", err)
	}
	log.Printf("log : %v", config)

	server(config)
}

func set_logfile(config *Config) {
	var writer io.Writer
	if config.log != "" {
		logFile, err := os.OpenFile(config.log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Error opening log file: %v", err)
		}
		defer logFile.Close()
		writer = io.MultiWriter(os.Stderr, logFile)
	} else {
		writer = os.Stderr
	}
	log.SetOutput(writer)
	log.SetFlags(log.Ltime | log.Lshortfile)
}
