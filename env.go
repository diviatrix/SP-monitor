package main

import (
	"os"
	"time"
)

// EnvConfig holds environment-derived runtime settings.
type EnvConfig struct {
	ExportPath     string
	ExportName     string
	ImportPath     string
	ImportName     string
	StatusInterval time.Duration
	DialTimeout    time.Duration
}

// LoadEnv reads environment variables and applies defaults.
// Defaults:
//
//	EXPORT_PATH      -> "."
//	EXPORT_NAME      -> "status.json"
//	IMPORT_PATH      -> EXPORT_PATH
//	IMPORT_NAME      -> EXPORT_NAME
//	STATUS_INTERVAL  -> "5s" (time.Duration)
//	PORT_DIAL_TIMEOUT-> "200ms" (time.Duration)
func LoadEnv() EnvConfig {
	exportPath := os.Getenv("EXPORT_PATH")
	if exportPath == "" {
		exportPath = "."
	}

	exportName := os.Getenv("EXPORT_NAME")
	if exportName == "" {
		exportName = "status.json"
	}

	importPath := os.Getenv("IMPORT_PATH")
	if importPath == "" {
		importPath = exportPath
	}

	importName := os.Getenv("IMPORT_NAME")
	if importName == "" {
		importName = exportName
	}

	intervalStr := os.Getenv("STATUS_INTERVAL")
	statusInterval := 5 * time.Second
	if intervalStr != "" {
		if dur, err := time.ParseDuration(intervalStr); err == nil && dur > 0 {
			statusInterval = dur
		}
	}

	dialStr := os.Getenv("PORT_DIAL_TIMEOUT")
	dialTimeout := 200 * time.Millisecond
	if dialStr != "" {
		if dur, err := time.ParseDuration(dialStr); err == nil && dur > 0 {
			dialTimeout = dur
		}
	}

	return EnvConfig{
		ExportPath:     exportPath,
		ExportName:     exportName,
		ImportPath:     importPath,
		ImportName:     importName,
		StatusInterval: statusInterval,
		DialTimeout:    dialTimeout,
	}
}
