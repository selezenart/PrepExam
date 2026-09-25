// Package store holds the user's configuration and their saved progress.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const configFile = "config.yaml"

// Config holds the exam rules. They are configurable so a variant ruleset can
// be tried without a rebuild, but the defaults are the real exam's.
type Config struct {
	ExamDuration      time.Duration
	PointsPerExercise int
	PassMark          int
}

// DefaultConfig returns the real Exam Rank 02 rules: three hours, 25 points
// per exercise, 100 points to pass, which means clearing all four levels.
func DefaultConfig() Config {
	return Config{
		ExamDuration:      3 * time.Hour,
		PointsPerExercise: 25,
		PassMark:          100,
	}
}

// configFileFormat is the on-disk shape, kept separate so the duration can be
// written the way a person would type it.
type configFileFormat struct {
	ExamDuration      string `yaml:"exam_duration"`
	PointsPerExercise int    `yaml:"points_per_exercise"`
	PassMark          int    `yaml:"pass_mark"`
}

// LoadConfig reads config.yaml from dir, falling back to the defaults when it
// is absent. A malformed file is an error rather than a silent fallback:
// having quietly ignored the rules someone wrote would be worse than saying so.
func LoadConfig(dir string) (Config, error) {
	c := DefaultConfig()
	raw, err := os.ReadFile(filepath.Join(dir, configFile))
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, fmt.Errorf("reading %s: %w", configFile, err)
	}
	var f configFileFormat
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return c, fmt.Errorf("parsing %s: %w", configFile, err)
	}
	if f.ExamDuration != "" {
		d, err := time.ParseDuration(f.ExamDuration)
		if err != nil {
			return c, fmt.Errorf("exam_duration %q: %w", f.ExamDuration, err)
		}
		c.ExamDuration = d
	}
	if f.PointsPerExercise > 0 {
		c.PointsPerExercise = f.PointsPerExercise
	}
	if f.PassMark > 0 {
		c.PassMark = f.PassMark
	}
	return c, nil
}
