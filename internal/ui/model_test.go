package ui

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"exam02/internal/catalog"
	"exam02/internal/grader"
	"exam02/internal/session"
	"exam02/internal/store"
)

func testCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()
	fsys := fstest.MapFS{}
	for _, name := range []string{"ft_strlen", "ft_atoi", "epur_str", "ft_split"} {
		level := map[string]int{"ft_strlen": 1, "ft_atoi": 2, "epur_str": 3, "ft_split": 4}[name]
		dir := "exercises/" + name + "/"
		fsys[dir+"meta.yaml"] = &fstest.MapFile{Data: []byte(
			"name: " + name + "\nlevel: " + string(rune('0'+level)) + "\nkind: program\n" +
				"expected_file: " + name + ".c\nallowed_functions: []\nprototype: \"\"\n")}
		fsys[dir+"subject.md"] = &fstest.MapFile{Data: []byte("subject for " + name)}
		fsys[dir+"reference.c"] = &fstest.MapFile{Data: []byte("int main(void){return 0;}")}
		fsys[dir+"cases.txt"] = &fstest.MapFile{Data: []byte("{\"args\": []}\n")}
	}
	c, err := catalog.Load(fsys)
	if err != nil {
		t.Fatalf("catalog.Load() error = %v", err)
	}
	return c
}

func newTestModel(t *testing.T) Model {
	t.Helper()
	state, _ := store.Load(t.TempDir())
	return New(Deps{
		Catalog:  testCatalog(t),
		Grader:   grader.New(),
		Config:   store.DefaultConfig(),
		State:    state,
		StateDir: t.TempDir(),
		Root:     t.TempDir(),
	})
}

func key(s string) tea.KeyMsg {
	if len(s) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
	return tea.KeyMsg{Type: tea.KeyEnter}
}

func TestMenuIsTheFirstScreen(t *testing.T) {
	m := newTestModel(t)
	view := m.View()
	for _, want := range []string{"Exam", "Practice", "Quit"} {
		if !strings.Contains(view, want) {
			t.Errorf("menu does not offer %q:\n%s", want, view)
		}
	}
}

func TestQuittingFromTheMenu(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(key("q"))
	if cmd == nil {
		t.Fatal("pressing q returned no command, want tea.Quit")
	}
	if msg := cmd(); msg == nil {
		t.Error("the command produced no message, want a quit message")
	}
}

func TestStartingAnExamShowsTheSubjectAndTheClock(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(key("1"))
	view := next.View()
	if !strings.Contains(view, "subject for") {
		t.Errorf("the exam screen does not show a subject:\n%s", view)
	}
	if !strings.Contains(view, "Level 1") {
		t.Errorf("the exam screen does not show the level:\n%s", view)
	}
	if !strings.Contains(view, "0 / 100") {
		t.Errorf("the exam screen does not show points against the pass mark:\n%s", view)
	}
}

func TestPracticeListsEveryExercise(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(key("2"))
	view := next.View()
	for _, want := range []string{"ft_strlen", "ft_atoi", "epur_str", "ft_split"} {
		if !strings.Contains(view, want) {
			t.Errorf("the practice list is missing %q:\n%s", want, view)
		}
	}
}

func TestExamScreenRendersTheRemainingTime(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(key("1"))
	// Three hours, so the clock should read close to it on the first frame.
	if !strings.Contains(next.View(), "2:59") && !strings.Contains(next.View(), "3:00") {
		t.Errorf("the exam screen does not show a countdown:\n%s", next.View())
	}
}

func TestTickAdvancesTheClockWithoutEndingTheExam(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(key("1"))
	after, cmd := next.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Error("a tick did not schedule the next one; the clock would stop")
	}
	if !strings.Contains(after.View(), "Level 1") {
		t.Errorf("the exam screen was lost on a tick:\n%s", after.View())
	}
}
func TestPracticeShowsPerExerciseStats(t *testing.T) {
	state, _ := store.Load(t.TempDir())
	state.Record("ft_strlen", "OK", true)
	state.Record("ft_atoi", "wrong output", false)
	m := New(Deps{
		Catalog: testCatalog(t), Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: t.TempDir(), Root: t.TempDir(),
	})
	next, _ := m.Update(key("2"))
	view := next.View()
	if !strings.Contains(view, "1/1") {
		t.Errorf("the practice list does not show ft_strlen's record:\n%s", view)
	}
	if !strings.Contains(view, "0/1") {
		t.Errorf("the practice list does not show ft_atoi's record:\n%s", view)
	}
}

func TestPracticeCursorMoves(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(key("2"))
	down, _ := next.Update(tea.KeyMsg{Type: tea.KeyDown})
	if down.cursor != 1 {
		t.Errorf("cursor = %d after down, want 1", down.cursor)
	}
	up, _ := down.Update(tea.KeyMsg{Type: tea.KeyUp})
	if up.cursor != 0 {
		t.Errorf("cursor = %d after up, want 0", up.cursor)
	}
	// The cursor must not run off the top of the list.
	again, _ := up.Update(tea.KeyMsg{Type: tea.KeyUp})
	if again.cursor != 0 {
		t.Errorf("cursor = %d at the top, want it to stay at 0", again.cursor)
	}
}

func TestPracticeDrillPicksTheWeakest(t *testing.T) {
	state, _ := store.Load(t.TempDir())
	// Everything attempted once, ft_split failed the most.
	state.Record("ft_strlen", "OK", true)
	state.Record("ft_atoi", "OK", true)
	state.Record("epur_str", "OK", true)
	state.Record("ft_split", "wrong output", false)
	state.Record("ft_split", "wrong output", false)
	m := New(Deps{
		Catalog: testCatalog(t), Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: t.TempDir(), Root: t.TempDir(),
	})
	list, _ := m.Update(key("2"))
	drill, _ := list.Update(key("w"))
	if drill.current.Name != "ft_split" {
		t.Errorf("drill chose %q, want ft_split, the most failed", drill.current.Name)
	}
}

func TestResultScreenStatesPassOrFail(t *testing.T) {
	m := newTestModel(t)
	exam, _ := m.Update(key("1"))
	exam.screen = screenResult
	view := exam.View()
	if !strings.Contains(view, "0 / 100") {
		t.Errorf("the result screen does not show the score:\n%s", view)
	}
	if !strings.Contains(strings.ToLower(view), "fail") {
		t.Errorf("the result screen does not state the outcome:\n%s", view)
	}
}

func TestMenuOffersResumeOnlyWhenAnExamIsInFlight(t *testing.T) {
	m := newTestModel(t)
	if strings.Contains(m.View(), "Resume") {
		t.Errorf("the menu offers Resume with no exam in flight:\n%s", m.View())
	}

	pools := map[int][]string{1: {"ft_strlen"}, 2: {"ft_atoi"}, 3: {"epur_str"}, 4: {"ft_split"}}
	exam, err := session.NewExam(store.DefaultConfig(), pools, rand.New(rand.NewSource(1)), time.Now())
	if err != nil {
		t.Fatalf("NewExam() error = %v", err)
	}
	state, _ := store.Load(t.TempDir())
	withExam := New(Deps{
		Catalog: testCatalog(t), Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: t.TempDir(), Root: t.TempDir(), ResumeExam: exam,
	})
	if !strings.Contains(withExam.View(), "Resume") {
		t.Errorf("the menu does not offer Resume for an exam in flight:\n%s", withExam.View())
	}
}

func TestResumingReturnsToTheExamInProgress(t *testing.T) {
	pools := map[int][]string{1: {"ft_strlen"}, 2: {"ft_atoi"}, 3: {"epur_str"}, 4: {"ft_split"}}
	exam, err := session.NewExam(store.DefaultConfig(), pools, rand.New(rand.NewSource(1)), time.Now())
	if err != nil {
		t.Fatalf("NewExam() error = %v", err)
	}
	exam.Record(true, time.Now()) // already cleared level 1
	state, _ := store.Load(t.TempDir())
	m := New(Deps{
		Catalog: testCatalog(t), Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: t.TempDir(), Root: t.TempDir(), ResumeExam: exam,
	})
	next, _ := m.Update(key("c"))
	view := next.View()
	if !strings.Contains(view, "Level 2") {
		t.Errorf("resuming did not return to level 2:\n%s", view)
	}
	if !strings.Contains(view, "25 / 100") {
		t.Errorf("resuming lost the score:\n%s", view)
	}
}

func TestQuittingAnExamReturnsToTheMenuWithResumeOffered(t *testing.T) {
	m := newTestModel(t)
	exam, _ := m.Update(key("1"))
	back, _ := exam.Update(key("q"))
	if back.screen != screenMenu {
		t.Fatalf("q during an exam went to screen %d, want the menu", back.screen)
	}
	if !strings.Contains(back.View(), "Resume") {
		t.Errorf("the menu does not offer to resume the exam just left:\n%s", back.View())
	}
}

func TestAPassInAnExamStaysVisibleOnTheNextExercise(t *testing.T) {
	m := newTestModel(t)
	exam, _ := m.Update(key("1"))
	next, _ := exam.Update(gradedMsg{verdict: grader.Verdict{Status: grader.StatusOK, Summary: "all cases passed"}})
	view := next.View()
	if !strings.Contains(view, "Level 2") {
		t.Fatalf("a pass did not advance to level 2:\n%s", view)
	}
	if !strings.Contains(view, "OK") {
		t.Errorf("the pass was not shown to the candidate:\n%s", view)
	}
}

func TestResumingAnnouncesASubstitutedExercise(t *testing.T) {
	pools := map[int][]string{1: {"ft_strlen"}, 2: {"ft_atoi"}, 3: {"epur_str"}, 4: {"ft_split"}}
	exam, err := session.NewExam(store.DefaultConfig(), pools, rand.New(rand.NewSource(1)), time.Now())
	if err != nil {
		t.Fatalf("NewExam() error = %v", err)
	}
	exam.CurrentExercise = "gone_from_catalog"
	raw, _ := session.Encode(exam)
	resumed, err := session.Decode(raw, pools, rand.New(rand.NewSource(1)))
	if err != nil || resumed.Substitution() == nil {
		t.Fatalf("Decode() = %v, %v; want a substitution", resumed, err)
	}
	state, _ := store.Load(t.TempDir())
	m := New(Deps{
		Catalog: testCatalog(t), Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: t.TempDir(), Root: t.TempDir(), ResumeExam: resumed,
	})
	next, _ := m.Update(key("c"))
	view := next.View()
	if !strings.Contains(view, "gone_from_catalog") || !strings.Contains(view, "ft_strlen") {
		t.Errorf("the exam screen does not say the exercise was swapped:\n%s", view)
	}
}

func TestAGradingErrorIsShownOnTheExerciseScreen(t *testing.T) {
	m := newTestModel(t)
	exam, _ := m.Update(key("1"))
	next, _ := exam.Update(gradedMsg{err: errors.New("cc not found")})
	if !strings.Contains(next.View(), "cc not found") {
		t.Errorf("a grading error was swallowed:\n%s", next.View())
	}
}

func TestPracticeListScrollsToKeepTheCursorOnScreen(t *testing.T) {
	fsys := fstest.MapFS{}
	for i := 0; i < 40; i++ {
		dir := fmt.Sprintf("exercises/ex_%02d/", i)
		fsys[dir+"meta.yaml"] = &fstest.MapFile{Data: []byte(fmt.Sprintf(
			"name: ex_%02d\nlevel: 1\nkind: program\nexpected_file: ex_%02d.c\nallowed_functions: []\nprototype: \"\"\n", i, i))}
		fsys[dir+"subject.md"] = &fstest.MapFile{Data: []byte("s")}
		fsys[dir+"reference.c"] = &fstest.MapFile{Data: []byte("int main(void){return 0;}")}
	}
	c, err := catalog.Load(fsys)
	if err != nil {
		t.Fatalf("catalog.Load() error = %v", err)
	}
	state, _ := store.Load(t.TempDir())
	m := New(Deps{
		Catalog: c, Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: t.TempDir(), Root: t.TempDir(),
	})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m, _ = m.Update(key("2"))
	for i := 0; i < 30; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	view := m.View()
	if lines := strings.Count(view, "\n") + 1; lines > 20 {
		t.Errorf("the practice list is %d lines tall on a 20 line terminal", lines)
	}
	if !strings.Contains(view, "ex_30") {
		t.Errorf("the selected exercise ex_30 is off screen:\n%s", view)
	}
	if !strings.Contains(view, "enter") {
		t.Errorf("the key hints scrolled away:\n%s", view)
	}
}
