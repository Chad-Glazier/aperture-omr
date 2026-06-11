package extractor

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadTemplate(path string) (Template, error) {
	var tmp Template

	data, err := os.ReadFile(path)
	if err != nil {
		return tmp, fmt.Errorf("failed to read template file: %w", err)
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return tmp, fmt.Errorf("failed to parse template JSON: %w", err)
	}

	return tmp, nil
}
