package csvs

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/activities"
)

const (
	csvValidatorCmd     = "csv-validator-cmd"
	validationErrorCode = 3
)

// CSVValidatorCmd validates a CSV file by running csv-validator-cmd.
type CSVValidatorCmd struct {
	Command string
}

var _ activities.MetadataValidator = (*CSVValidatorCmd)(nil)

func NewCSVValidatorCmd() *CSVValidatorCmd {
	return &CSVValidatorCmd{Command: csvValidatorCmd}
}

func (c *CSVValidatorCmd) Validate(ctx context.Context, csvPath, schemaPath string) ([]string, error) {
	command := c.Command
	if command == "" {
		command = csvValidatorCmd
	}

	cmd := exec.CommandContext(ctx, command, "--skip-file-checks", csvPath, schemaPath) // #nosec G204
	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("%s not found: %w", command, err)
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == validationErrorCode {
			validationErrors := parseCSVValidatorOutput(string(out))
			if len(validationErrors) > 0 {
				return validationErrors, nil
			}
		}

		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s failed: %s", command, msg)
	}

	return parseCSVValidatorOutput(string(out)), nil
}

func parseCSVValidatorOutput(output string) []string {
	var errs []string
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		switch line {
		case "", "FAIL":
			continue
		case "PASS":
			return nil
		}
		if strings.Contains(line, "processing") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "Error:"))
		if line != "" {
			errs = append(errs, line)
		}
	}
	return errs
}
