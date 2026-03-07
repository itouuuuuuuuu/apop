package selector

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/itouuuuuuuuu/apop/internal/profile"
)

func Select(profiles []profile.Profile) (string, error) {
	if _, err := exec.LookPath("fzf"); err != nil {
		return fallbackSelect(profiles)
	}
	return fzfSelect(profiles)
}

func fzfSelect(profiles []profile.Profile) (string, error) {
	current := profile.CurrentProfile()

	var items []string
	for _, p := range profiles {
		items = append(items, p.Name)
	}

	cmd := exec.Command("fzf",
		"--height", "40%",
		"--reverse",
		"--header", fmt.Sprintf("Current: %s | Select AWS Profile", current),
	)

	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to start fzf: %w", err)
	}

	go func() {
		defer stdin.Close()
		for _, item := range items {
			fmt.Fprintln(stdin, item)
		}
	}()

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("profile selection cancelled")
	}

	return strings.TrimSpace(string(out)), nil
}

func fallbackSelect(profiles []profile.Profile) (string, error) {
	current := profile.CurrentProfile()
	fmt.Fprintf(os.Stderr, "Current: %s\n", current)
	fmt.Fprintf(os.Stderr, "Select AWS Profile:\n\n")

	for i, p := range profiles {
		marker := "  "
		if p.Name == current {
			marker = "> "
		}
		fmt.Fprintf(os.Stderr, "%s%d) %s\n", marker, i+1, p.Name)
	}

	fmt.Fprintf(os.Stderr, "\nEnter number: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("input cancelled")
	}

	var idx int
	if _, err := fmt.Sscanf(scanner.Text(), "%d", &idx); err != nil || idx < 1 || idx > len(profiles) {
		return "", fmt.Errorf("invalid selection")
	}

	return profiles[idx-1].Name, nil
}
