package session

import (
	"math/rand"
	"testing"
	"time"

	"exam02/internal/store"
)

func testPools() map[int][]string {
	return map[int][]string{
		1: {"ft_strlen", "rot_13"},
		2: {"ft_atoi"},
		3: {"epur_str"},
		4: {"ft_split"},
	}
}

func newTestExam(t *testing.T, start time.Time) *Exam {
	t.Helper()
	e, err := NewExam(store.DefaultConfig(), testPools(), rand.New(rand.NewSource(1)), start)
	if err != nil {
		t.Fatalf("NewExam() error = %v", err)
	}
	return e
}

func TestNewExamStartsAtLevelOne(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	if e.Level() != 1 {
		t.Errorf("Level() = %d, want 1", e.Level())
	}
	if e.Points() != 0 {
		t.Errorf("Points() = %d, want 0", e.Points())
	}
	if got := e.Current(); got != "ft_strlen" && got != "rot_13" {
		t.Errorf("Current() = %q, want one of the level 1 pool", got)
	}
}

func TestPassingAdvancesALevelAndScores(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	e.Record(true, start.Add(time.Minute))
	if e.Level() != 2 {
		t.Errorf("Level() = %d, want 2", e.Level())
	}
	if e.Points() != 25 {
		t.Errorf("Points() = %d, want 25", e.Points())
	}
	if e.Current() != "ft_atoi" {
		t.Errorf("Current() = %q, want ft_atoi", e.Current())
	}
}

func TestFailingRedrawsFromTheSameLevel(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	e.Record(false, start.Add(time.Minute))
	if e.Level() != 1 {
		t.Errorf("Level() = %d, want 1", e.Level())
	}
	if e.Points() != 0 {
		t.Errorf("Points() = %d, want 0", e.Points())
	}
	if e.Over(start.Add(2 * time.Minute)) {
		t.Error("Over() = true; a failed attempt must not end the exam")
	}
}

func TestClearingAllFourLevelsPasses(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	for i := 0; i < 4; i++ {
		e.Record(true, start.Add(time.Duration(i)*time.Minute))
	}
	if e.Points() != 100 {
		t.Errorf("Points() = %d, want 100", e.Points())
	}
	if !e.Passed() {
		t.Error("Passed() = false, want true at 100 points")
	}
	if !e.Over(start.Add(5 * time.Minute)) {
		t.Error("Over() = false, want true after the last level")
	}
	if e.Current() != "" {
		t.Errorf("Current() = %q, want empty once the exam is over", e.Current())
	}
}

func TestThreeOfFourLevelsFails(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	for i := 0; i < 3; i++ {
		e.Record(true, start.Add(time.Duration(i)*time.Minute))
	}
	if e.Points() != 75 {
		t.Errorf("Points() = %d, want 75", e.Points())
	}
	if e.Passed() {
		t.Error("Passed() = true at 75 points, want false: the pass mark is 100")
	}
}

func TestTimeExpiryEndsTheExam(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	if e.Over(start.Add(2 * time.Hour)) {
		t.Error("Over() = true after 2h, want false")
	}
	if !e.Over(start.Add(3 * time.Hour)) {
		t.Error("Over() = false after 3h, want true")
	}
	if got := e.Remaining(start.Add(4 * time.Hour)); got != 0 {
		t.Errorf("Remaining() = %v past the deadline, want 0", got)
	}
}

func TestRemainingCountsDown(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	if got := e.Remaining(start.Add(time.Hour)); got != 2*time.Hour {
		t.Errorf("Remaining() = %v, want 2h", got)
	}
}

func TestAttemptsAreRecordedInOrder(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	first := e.Current()
	e.Record(false, start.Add(time.Minute))
	e.Record(true, start.Add(2*time.Minute))

	got := e.Attempts()
	if len(got) != 2 {
		t.Fatalf("Attempts() = %d, want 2", len(got))
	}
	if got[0].Exercise != first || got[0].Passed {
		t.Errorf("first attempt = %+v, want %q failed", got[0], first)
	}
	if !got[1].Passed {
		t.Errorf("second attempt = %+v, want passed", got[1])
	}
}

func TestRecordAfterTheExamIsOverIsIgnored(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	for i := 0; i < 4; i++ {
		e.Record(true, start.Add(time.Duration(i)*time.Minute))
	}
	e.Record(true, start.Add(10*time.Minute))
	if e.Points() != 100 {
		t.Errorf("Points() = %d, want it capped at 100", e.Points())
	}
}

func TestNewExamNeedsEveryLevelStocked(t *testing.T) {
	pools := testPools()
	delete(pools, 3)
	if _, err := NewExam(store.DefaultConfig(), pools, rand.New(rand.NewSource(1)), time.Now()); err == nil {
		t.Fatal("NewExam() error = nil, want an error when a level has no exercises")
	}
}

func TestExamSurvivesASaveAndReload(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	e.Record(true, start.Add(time.Minute))    // clears level 1
	e.Record(false, start.Add(2*time.Minute)) // fails once at level 2

	encoded, err := Encode(e)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	restored, err := Decode(encoded, testPools(), rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if restored.Level() != 2 {
		t.Errorf("Level() = %d, want 2", restored.Level())
	}
	if restored.Points() != 25 {
		t.Errorf("Points() = %d, want 25", restored.Points())
	}
	if len(restored.Attempts()) != 2 {
		t.Errorf("Attempts() = %d, want 2", len(restored.Attempts()))
	}
	if restored.Current() == "" {
		t.Error("Current() is empty; the restored exam has nothing to work on")
	}
	if restored.Substitution() != nil {
		t.Errorf("Substitution() = %+v, want nil: nothing changed in the pools", restored.Substitution())
	}
}

func TestRestoredExamKeepsCountingFromItsOriginalStart(t *testing.T) {
	// The clock must not restart on resume, or quitting and reopening would
	// be a way to buy unlimited time.
	start := time.Now().Add(-time.Hour)
	e := newTestExam(t, start)
	encoded, err := Encode(e)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	restored, err := Decode(encoded, testPools(), rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got := restored.Remaining(time.Now()); got > 2*time.Hour {
		t.Errorf("Remaining() = %v, want about 2h: the clock must not restart", got)
	}
}

func TestDecodeOfAFinishedExamIsNotResumable(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	for i := 0; i < 4; i++ {
		e.Record(true, start.Add(time.Duration(i)*time.Minute))
	}
	encoded, err := Encode(e)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	restored, err := Decode(encoded, testPools(), rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if !restored.Over(time.Now()) {
		t.Error("Over() = false; a finished exam must not come back resumable")
	}
}

// --- Findings from the Task 9 review: Decode robustness and coverage. ---

func TestDecodeErrorsWhenTheCurrentLevelPoolIsNowEmpty(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	e.Record(true, start.Add(time.Minute)) // clears level 1; now on level 2

	encoded, err := Encode(e)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	pools := testPools()
	delete(pools, 2) // the catalog lost every exercise for level 2

	if _, err := Decode(encoded, pools, rand.New(rand.NewSource(2))); err == nil {
		t.Fatal("Decode() error = nil, want an error: level 2 has nothing to draw from")
	}
}

func TestDecodeErrorsWhenAFutureLevelPoolIsNowEmpty(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start) // still on level 1, hasn't reached level 4 yet

	encoded, err := Encode(e)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	pools := testPools()
	delete(pools, 4) // level 4 is emptied before the candidate ever gets there

	if _, err := Decode(encoded, pools, rand.New(rand.NewSource(2))); err == nil {
		t.Fatal("Decode() error = nil, want an error: level 4 could never be drawn from either")
	}
}

func TestDecodeRedrawsAndReportsAStaleExercise(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	e.Record(true, start.Add(time.Minute)) // clears level 1; current exercise is ft_atoi at level 2

	encoded, err := Encode(e)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	pools := testPools()
	pools[2] = []string{"ft_atoi_base"} // ft_atoi was renamed out of the catalog

	restored, err := Decode(encoded, pools, rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if restored.Current() != "ft_atoi_base" {
		t.Errorf("Current() = %q, want ft_atoi_base: the stale exercise must be redrawn", restored.Current())
	}
	sub := restored.Substitution()
	if sub == nil {
		t.Fatal("Substitution() = nil, want the swap reported so a caller can tell the candidate")
	}
	if sub.From != "ft_atoi" || sub.To != "ft_atoi_base" {
		t.Errorf("Substitution() = %+v, want {From: ft_atoi, To: ft_atoi_base}", *sub)
	}
}

func TestDecodeReattachesRandomnessForARedraw(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start) // level 1, pool of two: ft_strlen, rot_13

	encoded, err := Encode(e)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	restored, err := Decode(encoded, testPools(), rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Failing at level 1, whose pool has two entries, forces draw() through
	// rng.Intn. A regression that drops the rng half of Attach panics here.
	restored.Record(false, start.Add(time.Minute))
	if got := restored.Current(); got != "ft_strlen" && got != "rot_13" {
		t.Errorf("Current() = %q, want one of the level 1 pool", got)
	}
}

func TestDrawWithADuplicatePoolEntryDoesNotHang(t *testing.T) {
	start := time.Now()
	pools := testPools()
	pools[1] = []string{"dup", "dup"} // every level-1 entry is the same name

	e, err := NewExam(store.DefaultConfig(), pools, rand.New(rand.NewSource(1)), start)
	if err != nil {
		t.Fatalf("NewExam() error = %v", err)
	}
	if e.Current() != "dup" {
		t.Errorf("Current() = %q, want dup", e.Current())
	}
	e.Record(false, start.Add(time.Minute)) // must not spin looking for a different name
	if e.Current() != "dup" {
		t.Errorf("Current() = %q, want dup", e.Current())
	}
}

func TestNewExamCopiesThePoolsItIsGiven(t *testing.T) {
	start := time.Now()
	pools := testPools()

	e, err := NewExam(store.DefaultConfig(), pools, rand.New(rand.NewSource(1)), start)
	if err != nil {
		t.Fatalf("NewExam() error = %v", err)
	}

	pools[1][0] = "mutated" // the caller mutates the slice after handing it over
	delete(pools, 2)        // and removes a level entirely

	// Passing level 1 must still draw the real level-2 exercise, unaffected
	// by either mutation: NewExam must not alias the caller's map or slices.
	e.Record(true, start.Add(time.Minute))
	if e.Current() != "ft_atoi" {
		t.Errorf("Current() = %q, want ft_atoi", e.Current())
	}
}

func TestAttemptsReturnsACopy(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	e.Record(true, start.Add(time.Minute))

	got := e.Attempts()
	got[0].Passed = false // mutate the returned slice

	if !e.Attempts()[0].Passed {
		t.Error("Attempts() aliases internal state: mutating the returned slice changed it")
	}
}
