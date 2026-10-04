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

func TestToggleUnsureMarksAndUnmarksAndSurvivesASave(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir)

	// Marking does not need an attempt first: you can be unsure of an
	// exercise you have only read.
	if got := s.ToggleUnsure("rot_13"); !got {
		t.Fatalf("ToggleUnsure() = false, want true on first toggle")
	}
	if err := s.Save(dir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	again, _ := Load(dir)
	if !again.IsUnsure("rot_13") {
		t.Errorf("IsUnsure(rot_13) = false after reload, want true")
	}
	if st := again.Exercises["rot_13"]; st.Attempts != 0 {
		t.Errorf("Attempts = %d, want 0: marking is not an attempt", st.Attempts)
	}

	if got := again.ToggleUnsure("rot_13"); got {
		t.Errorf("ToggleUnsure() = true, want false on second toggle")
	}
	if again.IsUnsure("rot_13") {
		t.Errorf("IsUnsure(rot_13) = true, want false after unmarking")
	}
}

func TestUnsureSurvivesGradingAndUnmarkingKeepsTheRecord(t *testing.T) {
	s, _ := Load(t.TempDir())
	s.ToggleUnsure("ft_strlen")

	// A pass, perhaps a lucky one, must not clear the mark: only the
	// candidate knows whether they are confident now.
	s.Record("ft_strlen", "OK", true)
	if !s.IsUnsure("ft_strlen") {
		t.Errorf("IsUnsure = false after a pass, want the mark kept")
	}

	s.ToggleUnsure("ft_strlen")
	if st := s.Exercises["ft_strlen"]; st.Attempts != 1 || st.Passes != 1 {
		t.Errorf("Stats = %+v after unmarking, want the attempt kept", st)
	}
}

func TestStateWithoutUnsureFieldStillLoads(t *testing.T) {
	// state.json written by an earlier version has no "unsure" field.
	dir := t.TempDir()
	old := `{"exercises":{"rot_13":{"attempts":2,"passes":1,"last_status":"OK","last_at":"2026-09-25T10:00:00Z"}}}`
	if err := os.WriteFile(filepath.Join(dir, stateFile), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if s.IsUnsure("rot_13") {
		t.Errorf("IsUnsure(rot_13) = true, want false for a state file that predates marks")
	}
	if st := s.Exercises["rot_13"]; st.Attempts != 2 {
		t.Errorf("Attempts = %d, want 2", st.Attempts)
	}
}

func TestIsUnsureOfAnUnknownExerciseIsFalse(t *testing.T) {
	s, _ := Load(t.TempDir())
	if s.IsUnsure("never_seen") {
		t.Errorf("IsUnsure(never_seen) = true, want false")
	}
}

func TestOldestUnsurePrefersNeverAttemptedThenLeastRecent(t *testing.T) {
	s, _ := Load(t.TempDir())
	pool := []string{"a", "b", "c", "d"}

	if got := s.OldestUnsure(pool); got != "" {
		t.Errorf("OldestUnsure() = %q with nothing marked, want \"\"", got)
	}

	old := time.Now().Add(-72 * time.Hour)
	s.Exercises["a"] = &Stats{Attempts: 1, LastAt: time.Now(), Unsure: true}
	s.Exercises["b"] = &Stats{Attempts: 1, LastAt: old, Unsure: true}
	s.Exercises["c"] = &Stats{Attempts: 9, LastAt: old.Add(-time.Hour)} // older, but not marked
	if got := s.OldestUnsure(pool); got != "b" {
		t.Errorf("OldestUnsure() = %q, want b: the marked one attempted longest ago", got)
	}

	// Marked but never attempted comes before anything attempted.
	s.Exercises["d"] = &Stats{Unsure: true}
	if got := s.OldestUnsure(pool); got != "d" {
		t.Errorf("OldestUnsure() = %q, want d: marked and never attempted", got)
	}

	// Only names in the pool count.
	if got := s.OldestUnsure([]string{"a", "c"}); got != "a" {
		t.Errorf("OldestUnsure(a,c) = %q, want a", got)
	}
}
