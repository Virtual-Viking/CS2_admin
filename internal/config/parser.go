package config

import (
	"bufio"
	"os"
	"sort"
	"strings"
)

// ReadCfgFile reads a CS2 .cfg file (e.g. server.cfg) and parses lines in format
// `key "value"` or `key value`. Returns a map of key-value pairs.
// Ignores comments (lines starting with //). Handles quoted and unquoted values.
func ReadCfgFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		key, value, ok := parseCfgLine(line)
		if !ok {
			continue
		}

		if key != "" {
			result[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// parseCfgLine parses a single config line in format `key "value"` or `key value`.
// Returns (key, value, ok).
func parseCfgLine(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", "", false
	}

	// Find first space to split key from value
	idx := strings.Index(line, " ")
	if idx < 0 {
		// No space: treat whole line as key with empty value
		return line, "", true
	}

	key := strings.TrimSpace(line[:idx])
	valuePart := strings.TrimSpace(line[idx+1:])

	if key == "" {
		return "", "", false
	}

	// Parse value - quoted or unquoted
	var value string
	if strings.HasPrefix(valuePart, `"`) {
		// Quoted value: find closing quote
		valuePart = valuePart[1:]
		endIdx := strings.Index(valuePart, `"`)
		if endIdx >= 0 {
			value = valuePart[:endIdx]
		} else {
			value = valuePart
		}
	} else {
		// Unquoted: take until end of line or next space for simplicity
		// CS2 config values can have spaces in unquoted form, so we take rest of line
		value = valuePart
	}

	return key, value, true
}

// WriteCfgFile writes a map of cvars to a .cfg file in `key "value"` format,
// one per line. Keys are sorted alphabetically.
func WriteCfgFile(path string, cvars map[string]string) error {
	if cvars == nil {
		cvars = make(map[string]string)
	}

	keys := make([]string, 0, len(cvars))
	for k := range cvars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		v := cvars[k]
		sb.WriteString(k)
		sb.WriteString(" \"")
		sb.WriteString(escapeCfgValue(v))
		sb.WriteString("\"\n")
	}

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func escapeCfgValue(v string) string {
	return strings.ReplaceAll(v, `"`, `\"`)
}

// ReadMapcycle reads a mapcycle.txt file and returns a list of map names,
// one per line. Ignores empty lines and comments.
func ReadMapcycle(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var maps []string
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		maps = append(maps, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return maps, nil
}

// WriteMapcycle writes map names to a file, one per line.
func WriteMapcycle(path string, maps []string) error {
	var sb strings.Builder
	for _, m := range maps {
		if m != "" {
			sb.WriteString(m)
			sb.WriteString("\n")
		}
	}
	return os.WriteFile(path, []byte(sb.String()), 0644)
}
