package config

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

// LogLevel used for logger configuration
type LogLevel struct {
	Level zerolog.Level
}

// String returns log level as string
func (l *LogLevel) String() string {
	return l.Level.String()
}

// Set validates and sets the log level from string
func (l *LogLevel) Set(value string) error {
	level, err := zerolog.ParseLevel(strings.ToLower(value))
	if err != nil {
		return fmt.Errorf("invalid log level: %v", err)
	}
	l.Level = level
	return nil
}
