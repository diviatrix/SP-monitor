package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func loadServicesConfig(filename string) (*ServicesConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var sc ServicesConfig
	if err := json.Unmarshal(data, &sc); err != nil {
		// Print a short prefix of the file to help diagnose malformed JSON
		preview := string(data)
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		log.Printf("services.json parse error: %v\nfile: %s\npreview: %q", err, filename, preview)
		return nil, fmt.Errorf("%w", err)
	}
	return &sc, nil
}
