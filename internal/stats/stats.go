package stats

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/dropseed/git-stats/internal/config"
	"github.com/dropseed/git-stats/internal/git"
	"github.com/dropseed/git-stats/internal/notes"
)

var sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

type CommitStat struct {
	Type         string
	commitOrder  []string
	CommitValues map[string]float64
}

func NewCommitStat(typ string) *CommitStat {
	return &CommitStat{
		Type:         typ,
		CommitValues: make(map[string]float64),
	}
}

func (cs *CommitStat) ParseValue(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case string:
		v = strings.TrimSpace(v)
		if cs.Type == "%" {
			v = strings.TrimRight(v, "%")
		}
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot parse value %v", value)
	}
}

func (cs *CommitStat) AddOrUpdate(commit string, value any) {
	parsed, err := cs.ParseValue(value)
	if err != nil {
		fmt.Printf("Could not parse value %q on commit %s. Setting value to \"missing\".\n", value, commit)
		return
	}

	if _, exists := cs.CommitValues[commit]; !exists {
		cs.commitOrder = append(cs.commitOrder, commit)
	}
	cs.CommitValues[commit] = parsed
}

func (cs *CommitStat) Get(commit string) (float64, bool) {
	v, ok := cs.CommitValues[commit]
	return v, ok
}

func (cs *CommitStat) Values() []float64 {
	vals := make([]float64, 0, len(cs.commitOrder))
	for _, c := range cs.commitOrder {
		vals = append(vals, cs.CommitValues[c])
	}
	return vals
}

func (cs *CommitStat) Min() float64 {
	vals := cs.Values()
	if len(vals) == 0 {
		return 0
	}
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func (cs *CommitStat) Max() float64 {
	vals := cs.Values()
	if len(vals) == 0 {
		return 0
	}
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func (cs *CommitStat) Avg() float64 {
	vals := cs.Values()
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return math.Round(sum/float64(len(vals))*100) / 100
}

func (cs *CommitStat) FormatValue(v float64) string {
	if v == math.Trunc(v) {
		if cs.Type == "%" {
			return fmt.Sprintf("%d%%", int(v))
		}
		return fmt.Sprintf("%d", int(v))
	}
	if cs.Type == "%" {
		return fmt.Sprintf("%g%%", v)
	}
	return fmt.Sprintf("%g", v)
}

type CommitStats struct {
	Commits []string
	Keys    []string
	Stats   map[string]*CommitStat
}

func New() *CommitStats {
	return &CommitStats{
		Stats: make(map[string]*CommitStat),
	}
}

func (s *CommitStats) AppendCommit(commit string) {
	s.Commits = append(s.Commits, commit)
}

func (s *CommitStats) AddOrUpdate(commit, key string, value any, typ string) {
	if _, ok := s.Stats[key]; !ok {
		s.Stats[key] = NewCommitStat(typ)
		s.Keys = append(s.Keys, key)
	}
	s.Stats[key].AddOrUpdate(commit, value)
}

func (s *CommitStats) CommitHasStat(commit, key string) bool {
	stat, ok := s.Stats[key]
	if !ok {
		return false
	}
	_, has := stat.CommitValues[commit]
	return has
}

func (s *CommitStats) CommitsMissingStats(keys []string) []string {
	var missing []string
	for _, commit := range s.Commits {
		for _, key := range keys {
			if !s.CommitHasStat(commit, key) {
				missing = append(missing, commit)
				break
			}
		}
	}
	return missing
}

func (s *CommitStats) Print(valuesOnly bool, sep string) {
	if !valuesOnly {
		fmt.Print("commit")
		for _, key := range s.Keys {
			fmt.Print(sep + key)
		}
		fmt.Println()
	}

	for _, commit := range s.Commits {
		first := true
		if !valuesOnly {
			fmt.Print(commit)
			first = false
		}
		for _, key := range s.Keys {
			if !first {
				fmt.Print(sep)
			}
			first = false
			stat := s.Stats[key]
			v, ok := stat.Get(commit)
			if ok {
				fmt.Print(stat.FormatValue(v))
			}
		}
		fmt.Println()
	}
}

func (s *CommitStats) PrintTable(valuesOnly bool) {
	// Calculate column widths
	commitWidth := 7 // short hash length
	if valuesOnly {
		commitWidth = 0
	}

	colWidths := make([]int, len(s.Keys))
	for i, key := range s.Keys {
		colWidths[i] = len(key)
		stat := s.Stats[key]
		for _, commit := range s.Commits {
			v, ok := stat.Get(commit)
			if ok {
				w := len(stat.FormatValue(v))
				if w > colWidths[i] {
					colWidths[i] = w
				}
			}
		}
	}

	// Print header
	if !valuesOnly {
		fmt.Printf("%-*s", commitWidth, "commit")
		for i, key := range s.Keys {
			fmt.Printf("  %*s", colWidths[i], key)
		}
		fmt.Println()
	}

	// Print rows
	for _, commit := range s.Commits {
		if !valuesOnly {
			fmt.Printf("%.7s", commit)
		}
		for i, key := range s.Keys {
			stat := s.Stats[key]
			v, ok := stat.Get(commit)
			if ok {
				fmt.Printf("  %*s", colWidths[i], stat.FormatValue(v))
			} else {
				fmt.Printf("  %*s", colWidths[i], "")
			}
		}
		fmt.Println()
	}
}

func (s *CommitStats) Sparklines() {
	for _, key := range s.Keys {
		stat := s.Stats[key]
		fmt.Printf("\033[1m%s\033[0m (min %s, max %s, avg %s)\n",
			key,
			stat.FormatValue(stat.Min()),
			stat.FormatValue(stat.Max()),
			stat.FormatValue(stat.Avg()),
		)
		fmt.Println(sparkline(stat.Values()))
		fmt.Println()
	}
}

func (s *CommitStats) PrintPretty(valuesOnly bool) {
	s.PrintTable(valuesOnly)
	if len(s.Keys) > 0 {
		fmt.Println()
		s.Sparklines()
	}
}

func sparkline(values []float64) string {
	if len(values) == 0 {
		return ""
	}

	min, max := values[0], values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	var sb strings.Builder
	for _, v := range values {
		idx := 0
		if max > min {
			idx = int((v - min) / (max - min) * float64(len(sparkBlocks)-1))
		}
		sb.WriteRune(sparkBlocks[idx])
	}
	return sb.String()
}

func Load(keys []string, cfg *config.Config, fillDefaults bool, gitLogArgs []string) (*CommitStats, error) {
	args := []string{
		"log",
		fmt.Sprintf("--show-notes=%s", notes.RefPath),
		"--format=COMMIT\n%H\n%N",
	}
	args = append(args, gitLogArgs...)

	out, err := git.Output(args...)
	if err != nil {
		return nil, err
	}

	s := New()
	var commit string
	expectHash := false

	for _, line := range strings.Split(out, "\n") {
		if line == "COMMIT" {
			expectHash = true
			continue
		}

		if expectHash {
			commit = line
			s.AppendCommit(commit)
			expectHash = false

			if fillDefaults {
				for _, key := range keys {
					s.AddOrUpdate(commit, key, cfg.DefaultForStat(key), cfg.TypeForStat(key))
				}
			}
			continue
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		found := false
		for _, k := range keys {
			if k == key {
				found = true
				break
			}
		}
		if found {
			s.AddOrUpdate(commit, key, value, cfg.TypeForStat(key))
		}
	}

	return s, nil
}
