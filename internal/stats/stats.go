package stats

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/dropseed/git-stats/internal/config"
	"github.com/dropseed/git-stats/internal/git"
	"github.com/dropseed/git-stats/internal/notes"
)

var sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// ANSI color helpers
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

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
	Commits   []string
	ShortHash map[string]string
	Subjects  map[string]string
	Keys      []string
	Stats     map[string]*CommitStat
}

func New() *CommitStats {
	return &CommitStats{
		ShortHash: make(map[string]string),
		Subjects:  make(map[string]string),
		Stats:     make(map[string]*CommitStat),
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
	commitWidth := len("commit")
	if valuesOnly {
		commitWidth = 0
	} else {
		for _, commit := range s.Commits {
			if w := len(s.ShortHash[commit]); w > commitWidth {
				commitWidth = w
			}
		}
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

	hasSubjects := len(s.Subjects) > 0

	// Print header
	if !valuesOnly {
		fmt.Printf("%s%-*s%s", colorBold, commitWidth, "commit", colorReset)
		for i, key := range s.Keys {
			fmt.Printf("  %s%*s%s", colorBold, colWidths[i], key, colorReset)
		}
		if hasSubjects {
			fmt.Printf("  %smessage%s", colorBold, colorReset)
		}
		fmt.Println()
	}

	// Precompute max per key for highlighting
	maxVals := make(map[string]float64)
	for _, key := range s.Keys {
		stat := s.Stats[key]
		if len(s.Commits) >= 3 && stat.Min() != stat.Max() {
			maxVals[key] = stat.Max()
		}
	}

	// Print rows
	for _, commit := range s.Commits {
		if !valuesOnly {
			fmt.Printf("%s%-*s%s", colorYellow, commitWidth, s.ShortHash[commit], colorReset)
		}
		for i, key := range s.Keys {
			stat := s.Stats[key]
			v, ok := stat.Get(commit)
			if ok {
				valColor := colorCyan
				if max, has := maxVals[key]; has && v == max {
					valColor = colorGreen + colorBold
				}
				fmt.Printf("  %s%*s%s", valColor, colWidths[i], stat.FormatValue(v), colorReset)
			} else {
				fmt.Printf("  %*s", colWidths[i], "")
			}
		}
		if hasSubjects {
			subject := s.Subjects[commit]
			if len(subject) > 50 {
				subject = subject[:47] + "..."
			}
			fmt.Printf("  %s%s%s", colorDim, subject, colorReset)
		}
		fmt.Println()
	}
}

func (s *CommitStats) Sparklines() {
	for _, key := range s.Keys {
		stat := s.Stats[key]

		// Trend indicator
		vals := stat.Values()
		trend := ""
		if len(vals) >= 2 {
			first := vals[0]
			last := vals[len(vals)-1]
			delta := last - first
			if first != 0 {
				pct := math.Abs(delta/first) * 100
				pctStr := fmt.Sprintf("%.0f%%", pct)
				if pct < 1 {
					pctStr = "<1%"
				}
				if delta > 0 {
					trend = fmt.Sprintf(" %s↑ %s%s", colorGreen, pctStr, colorReset)
				} else if delta < 0 {
					trend = fmt.Sprintf(" %s↓ %s%s", colorRed, pctStr, colorReset)
				} else {
					trend = fmt.Sprintf(" %s→%s", colorDim, colorReset)
				}
			} else if delta > 0 {
				trend = fmt.Sprintf(" %s↑%s", colorGreen, colorReset)
			} else if delta < 0 {
				trend = fmt.Sprintf(" %s↓%s", colorRed, colorReset)
			}
		}

		fmt.Printf("%s%s%s %s(min %s%s%s, max %s%s%s, avg %s%s%s)%s%s\n",
			colorBold, key, colorReset,
			colorDim,
			colorCyan, stat.FormatValue(stat.Min()), colorDim,
			colorCyan, stat.FormatValue(stat.Max()), colorDim,
			colorCyan, stat.FormatValue(stat.Avg()), colorDim,
			colorReset,
			trend,
		)
		fmt.Printf("%s%s%s\n", colorGreen, sparkline(vals), colorReset)
		fmt.Println()
	}
}

func (s *CommitStats) PrintPretty(valuesOnly bool) {
	s.PrintTable(valuesOnly)
	if len(s.Keys) > 0 && len(s.Commits) >= 3 {
		fmt.Println()
		s.Sparklines()
	}
}

func (s *CommitStats) PrintJSON() {
	type entry struct {
		Commit  string            `json:"commit"`
		Subject string            `json:"subject,omitempty"`
		Stats   map[string]string `json:"stats"`
	}

	entries := make([]entry, 0, len(s.Commits))
	for _, commit := range s.Commits {
		e := entry{
			Commit:  commit,
			Subject: s.Subjects[commit],
			Stats:   make(map[string]string),
		}
		for _, key := range s.Keys {
			stat := s.Stats[key]
			v, ok := stat.Get(commit)
			if ok {
				e.Stats[key] = stat.FormatValue(v)
			}
		}
		entries = append(entries, e)
	}

	data, _ := json.MarshalIndent(entries, "", "  ")
	fmt.Println(string(data))
}

func sparkline(values []float64) string {
	if len(values) == 0 {
		return ""
	}

	// Interpolate to target width if we have fewer points
	targetWidth := 60
	if len(values) >= targetWidth {
		targetWidth = len(values)
	}
	display := interpolate(values, targetWidth)

	min, max := display[0], display[0]
	for _, v := range display[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	var sb strings.Builder
	for _, v := range display {
		idx := 0
		if max > min {
			idx = int((v - min) / (max - min) * float64(len(sparkBlocks)-1))
		}
		sb.WriteRune(sparkBlocks[idx])
	}
	return sb.String()
}

// interpolate stretches values to targetLen using linear interpolation.
func interpolate(values []float64, targetLen int) []float64 {
	if len(values) == 0 || targetLen <= 0 {
		return values
	}
	if len(values) >= targetLen {
		return values
	}

	result := make([]float64, targetLen)
	for i := range result {
		srcPos := float64(i) * float64(len(values)-1) / float64(targetLen-1)
		srcIdx := int(srcPos)
		frac := srcPos - float64(srcIdx)

		if srcIdx >= len(values)-1 {
			result[i] = values[len(values)-1]
		} else {
			result[i] = values[srcIdx]*(1-frac) + values[srcIdx+1]*frac
		}
	}
	return result
}

func Load(keys []string, cfg *config.Config, fillDefaults bool, gitLogArgs []string) (*CommitStats, error) {
	args := []string{
		"log",
		fmt.Sprintf("--show-notes=%s", notes.RefPath),
		"--format=COMMIT\n%H\n%h\n%s\n%N",
	}
	args = append(args, gitLogArgs...)

	out, err := git.Output(args...)
	if err != nil {
		return nil, err
	}

	s := New()
	keySet := make(map[string]bool, len(keys))
	for _, k := range keys {
		keySet[k] = true
	}
	var commit string
	expectHash := false
	expectShortHash := false
	expectSubject := false

	for _, line := range strings.Split(out, "\n") {
		if line == "COMMIT" {
			expectHash = true
			continue
		}

		if expectHash {
			commit = line
			s.AppendCommit(commit)
			expectHash = false
			expectShortHash = true

			if fillDefaults {
				for _, key := range keys {
					s.AddOrUpdate(commit, key, cfg.DefaultForStat(key), cfg.TypeForStat(key))
				}
			}
			continue
		}

		if expectShortHash {
			s.ShortHash[commit] = line
			expectShortHash = false
			expectSubject = true
			continue
		}

		if expectSubject {
			s.Subjects[commit] = line
			expectSubject = false
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

		if keySet[key] {
			s.AddOrUpdate(commit, key, value, cfg.TypeForStat(key))
		}
	}

	return s, nil
}
