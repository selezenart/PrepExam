package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const stateFile = "state.json"

// Stats is what is remembered about one exercise.
type Stats struct {
	Attempts   int       `json:"attempts"`
	Passes     int       `json:"passes"`
	LastStatus string    `json:"last_status"`
	LastAt     time.Time `json:"last_at"`
}

// Failures is how many attempts did not pass.
func (s Stats) Failures() int { return s.Attempts - s.Passes }

// State is the persisted progress. ExamJSON carries an in-progress exam as
// opaque JSON so that store does not depend on the session package and the
// two can change independently.
type State struct {
	Exercises map[string]*Stats `json:"exercises"`
	Exam      json.RawMessage   `json:"exam,omitempty"`
}

// Dir returns the per-user directory holding config and state.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locating the user config directory: %w", err)
	}
	dir := filepath.Join(base, "exam02")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}
	return dir, nil
}

// Load reads the saved state from dir. A missing or unreadable file gives
// empty state rather than an error: losing statistics is a nuisance, but
// refusing to start the application over them would be worse.
func Load(dir string) (*State, error) {
	s := &State{Exercises: map[string]*Stats{}}
	raw, err := os.ReadFile(filepath.Join(dir, stateFile))
	if err != nil {
		return s, nil
	}
	if err := json.Unmarshal(raw, s); err != nil {
		return &State{Exercises: map[string]*Stats{}}, nil
	}
	if s.Exercises == nil {
		s.Exercises = map[string]*Stats{}
	}
	return s, nil
}

// Save writes the state to dir atomically: a crash partway through must not
// leave a truncated file where the next Load would find it.
func (s *State) Save(dir string) error {
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".state-*.tmp")
	if err != nil {
		return fmt.Errorf("creating a temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename has succeeded

	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return fmt.Errorf("writing state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing state: %w", err)
	}
	if err := os.Rename(tmpName, filepath.Join(dir, stateFile)); err != nil {
		return fmt.Errorf("replacing state: %w", err)
	}
	return nil
}

// Record notes one grading attempt.
func (s *State) Record(exercise, status string, passed bool) {
	st, ok := s.Exercises[exercise]
	if !ok {
		st = &Stats{}
		s.Exercises[exercise] = st
	}
	st.Attempts++
	if passed {
		st.Passes++
	}
	st.LastStatus = status
	st.LastAt = time.Now()
}

// Weakest picks the exercise from pool most worth practising: the one never
// attempted, else the one with the most failures, ties going to whichever was
// attempted longest ago. Returns "" for an empty pool.
func (s *State) Weakest(pool []string) string {
	var best string
	var bestFailures int
	var bestAt time.Time

	for _, name := range pool {
		st, seen := s.Exercises[name]
		if !seen || st.Attempts == 0 {
			return name
		}
		failures := st.Failures()
		if best == "" || failures > bestFailures ||
			(failures == bestFailures && st.LastAt.Before(bestAt)) {
			best, bestFailures, bestAt = name, failures, st.LastAt
		}
	}
	return best
}
