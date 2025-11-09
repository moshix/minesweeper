package main
/* minesweeper for 3270 terminals
   copyright 2025 by moshix
   all rights reserved
*/


import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds the server configuration
type Config struct {
	Port         int
	InstanceName string
}

// LoadConfig parses the mine.cnf confg
func LoadConfig(filename string) (*Config, error) {
	config := &Config{
		Port:         3270, // Default port
		InstanceName: "Minesweeper Server",
	}

	file, err := os.Open(filename)
	if err != nil {
		// If config file doesn't exist, use defaults
		return config, nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format at line %d: %s", lineNum, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
 // some plausibility checks
		switch strings.ToLower(key) {
		case "port":
			port, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid port number at line %d: %s", lineNum, value)
			}
			if port < 1 || port > 65535 {
				return nil, fmt.Errorf("port number out of range at line %d: %d", lineNum, port)
			}
			config.Port = port

		case "instance_name":
			config.InstanceName = value

		default:
			// Ignore unknown keys for forward compatibility
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	return config, nil
}
