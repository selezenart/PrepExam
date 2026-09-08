package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadFromAnEmptyDirectoryGivesEmptyState(t *testing.T) {
	s, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(s.Exercises) != 0 {
		t.Errorf("Exercises = %v, want empty", s.Exercises)
	}
	if s.Exam != nil {
		t.Errorf("Exam = %v, want nil", s.Exam)
	}
}

func TestRecordAndRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	s.Record("ft_strlen", "OK", true)
	s.Record("rot_13", "wrong output", false)
	s.Record("rot_13", "wrong output", false)
	if err := s.Save(dir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	again, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := again.Exercises["ft_strlen"]; got.Attempts != 1 || got.Passes != 1 {
		t.Errorf("ft_strlen = %+v, want 1 attempt 1 pass", got)
	}
	if got := again.Exercises["rot_13"]; got.Attempts != 2 || got.Passes != 0 {
		t.Errorf("rot_13 = %+v, want 2 attempts 0 passes", got)
	}
	if again.Exercises["rot_13"].LastStatus != "wrong output" {
		t.Errorf("LastStatus = %q, want %q", again.Exercises["rot_13"].LastStatus, "wrong output")
	}
}

func TestSaveIsAtomic(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir)
	s.Record("ft_strlen", "OK", true)
	if err := s.Save(dir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	// A crash mid-save must not leave a partial file behind under a name the
	// next Load would read.
	for _, e := range entries {
		if e.Name() != stateFile {
			t.Errorf("Save() left %q behind, want only %q", e.Name(), stateFile)
		}
	}
}

func TestLoadIgnoresCorruptState(t *testing.T) {
	// Losing statistics is annoying; refusing to start is worse.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, stateFile), []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v, want it to recover", err)
	}
	if len(s.Exercises) != 0 {
		t.Errorf("Exercises = %v, want empty", s.Exercises)
	}
}

func TestWeakestPicksMostFailedThenLeastRecent(t *testing.T) {
	s, _ := Load(t.TempDir())
	pool := []string{"a", "b", "c"}

	// Never attempted beats attempted: you cannot be weaker than untested.
	s.Exercises["a"] = &Stats{Attempts: 5, Passes: 1, LastAt: time.Now()}
	if got := s.Weakest(pool); got != "b" && got != "c" {
		t.Errorf("Weakest() = %q, want an unattempted exercise", got)
	}

	old := time.Now().Add(-72 * time.Hour)
	s.Exercises["b"] = &Stats{Attempts: 3, Passes: 3, LastAt: time.Now()}
	s.Exercises["c"] = &Stats{Attempts: 4, Passes: 0, LastAt: old}
	if got := s.Weakest(pool); got != "c" {
		t.Errorf("Weakest() = %q, want c: it has the most failures", got)
	}
}

func TestDefaultConfigMatchesTheRealExam(t *testing.T) {
	c := DefaultConfig()
	if c.ExamDuration != 3*time.Hour {
		t.Errorf("ExamDuration = %v, want 3h", c.ExamDuration)
	}
	if c.PointsPerExercise != 25 {
		t.Errorf("PointsPerExercise = %d, want 25", c.PointsPerExercise)
	}
	if c.PassMark != 100 {
		t.Errorf("PassMark = %d, want 100", c.PassMark)
	}
}

func TestLoadConfigFallsBackToDefaults(t *testing.T) {
	c, err := LoadConfig(t.TempDir())
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if c != DefaultConfig() {
		t.Errorf("LoadConfig() = %+v, want the defaults", c)
	}
}

func TestLoadConfigReadsOverrides(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, configFile),
		[]byte("exam_duration: 90m\npoints_per_exercise: 25\npass_mark: 50\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if c.ExamDuration != 90*time.Minute {
		t.Errorf("ExamDuration = %v, want 90m", c.ExamDuration)
	}
	if c.PassMark != 50 {
		t.Errorf("PassMark = %d, want 50", c.PassMark)
	}
}
