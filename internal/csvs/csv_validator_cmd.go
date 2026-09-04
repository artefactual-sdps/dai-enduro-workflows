package csvs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"go.artefactual.dev/tools/temporal"
)

const csvValidatorCmd = "csv-validator-cmd"

// CSVValidatorCmd validates a CSV file by running csv-validator-cmd.
type CSVValidatorCmd struct {
	Command string
}

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
		if errors.Is(err, exec.ErrNotFound) || isNotFoundError(err) {
			return nil, temporal.NewNonRetryableError(fmt.Errorf("%s not found: %w", command, err))
		}
		if _, ok := errors.AsType[*exec.ExitError](err); ok {
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

func isNotFoundError(err error) bool {
	var pathErr *os.PathError
	return errors.As(err, &pathErr) && errors.Is(pathErr.Err, os.ErrNotExist)
}
