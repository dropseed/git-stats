package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Exec runs a git command and returns combined stdout+stderr.
// Use for commands where stderr context is useful in error messages.
func Exec(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// Output runs a git command and returns only stdout.
// Use for commands where you need to parse the output.
func Output(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// RunShell runs a shell command via sh -c and returns stdout.
func RunShell(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("command %q: %w", command, err)
	}
	return strings.TrimSpace(string(out)), nil
}
