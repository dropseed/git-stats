package notes

import (
	"fmt"
	"strings"

	"github.com/dropseed/git-stats/internal/git"
)

const Ref = "git-stats"
const RefPath = "refs/notes/git-stats"

func Save(key, value, commitish string) error {
	statLine := fmt.Sprintf("%s: %s", key, value)

	existing, err := Get(commitish)
	if err != nil || existing == "" {
		_, err := git.Exec("notes", "--ref", Ref, "add", "--force", "--message", statLine, commitish)
		return err
	}

	lines := strings.Split(existing, "\n")
	var edited []string
	hadStat := false

	for _, line := range lines {
		if strings.HasPrefix(line, key+":") {
			edited = append(edited, statLine)
			hadStat = true
		} else {
			edited = append(edited, line)
		}
	}

	if !hadStat {
		edited = append(edited, statLine)
	}

	message := strings.Join(edited, "\n")
	if strings.TrimSpace(message) == strings.TrimSpace(existing) {
		return nil
	}

	_, err = git.Exec("notes", "--ref", Ref, "add", "--force", "--message", message, commitish)
	return err
}

func Get(commitish string) (string, error) {
	out, err := git.Output("notes", "--ref", Ref, "show", commitish)
	if err != nil {
		return "", err
	}
	return out, nil
}

func Show(commitish string) error {
	note, err := Get(commitish)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(note, "\n") {
		if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
			fmt.Printf("\033[1m%s:\033[0m\033[36m%s\033[0m\n", parts[0], parts[1])
		} else if line != "" {
			fmt.Println(line)
		}
	}
	return nil
}

func DeleteKey(key, commitish string) error {
	existing, err := Get(commitish)
	if err != nil {
		return err
	}

	var edited []string
	for _, line := range strings.Split(existing, "\n") {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, key+":") {
			edited = append(edited, line)
		}
	}

	if len(edited) > 0 {
		message := strings.Join(edited, "\n")
		_, err = git.Exec("notes", "--ref", Ref, "add", "--force", "--message", message, commitish)
		return err
	}

	_, err = git.Exec("notes", "--ref", Ref, "remove", commitish)
	return err
}

func Push() error {
	_, err := git.Exec("push", "origin", RefPath)
	return err
}

func Fetch(force bool) error {
	args := []string{"fetch"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, "origin", fmt.Sprintf("%s:%s", RefPath, RefPath))
	_, err := git.Exec(args...)
	return err
}

func Clear(remote bool) error {
	if remote {
		_, err := git.Exec("push", "--delete", "origin", RefPath)
		return err
	}
	_, err := git.Exec("update-ref", "-d", RefPath)
	return err
}
