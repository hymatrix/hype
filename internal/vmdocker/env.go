package vmdocker

import (
	"bufio"
	"strings"
)

func ParseEnvFile(path string, readFile func(string) ([]byte, error)) (map[string]string, error) {
	content, err := readFile(path)
	if err != nil {
		return nil, err
	}

	return ParseEnvContent(string(content))
}

func ParseEnvContent(content string) (map[string]string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	return parseEnvScanner(scanner)
}

func parseEnvScanner(scanner *bufio.Scanner) (map[string]string, error) {
	values := make(map[string]string)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" || value == "" {
			continue
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}
