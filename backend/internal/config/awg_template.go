package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type AWGTemplate struct {
	Jc   string
	Jmin string
	Jmax string
	S1   string
	S2   string
	H1   string
	H2   string
	H3   string
	H4   string
}

func LoadAWGTemplate(path string) (AWGTemplate, error) {
	file, err := os.Open(path)
	if err != nil {
		return AWGTemplate{}, fmt.Errorf("open template file: %w", err)
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return AWGTemplate{}, fmt.Errorf("invalid template line: %q", line)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return AWGTemplate{}, fmt.Errorf("read template file: %w", err)
	}

	required := []string{"Jc", "Jmin", "Jmax", "S1", "S2", "H1", "H2", "H3", "H4"}
	for _, key := range required {
		if values[key] == "" {
			return AWGTemplate{}, fmt.Errorf("missing %s in template file", key)
		}
	}

	return AWGTemplate{
		Jc:   values["Jc"],
		Jmin: values["Jmin"],
		Jmax: values["Jmax"],
		S1:   values["S1"],
		S2:   values["S2"],
		H1:   values["H1"],
		H2:   values["H2"],
		H3:   values["H3"],
		H4:   values["H4"],
	}, nil
}
