package ui

import (
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
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

func TestStartingAnExamOverAnInFlightOneAsksFirst(t *testing.T) {
	m := newTestModel(t)
	exam, _ := m.Update(key("1"))
	menu, _ := exam.Update(key("q"))
	first := menu.exam

	warned, _ := menu.Update(key("1"))
	if warned.screen != screenMenu || warned.exam != first {
		t.Fatal("one keypress discarded the exam in progress")
	}
	if !strings.Contains(warned.View(), "again") {
		t.Errorf("the menu does not explain how to confirm:\n%s", warned.View())
	}

	restarted, _ := warned.Update(key("1"))
	if restarted.screen != screenExam || restarted.exam == first {
		t.Error("confirming did not start a new exam")
	}
}

func TestTheExerciseScreenIsEnglishOnly(t *testing.T) {
	fsys := fstest.MapFS{
		"exercises/ft_strlen/meta.yaml": &fstest.MapFile{Data: []byte(
			"name: ft_strlen\nlevel: 1\nkind: program\nexpected_file: ft_strlen.c\nallowed_functions: []\nprototype: \"\"\n")},
		"exercises/ft_strlen/subject.md":  &fstest.MapFile{Data: []byte("subject for ft_strlen")},
		"exercises/ft_strlen/spanish.md":  &fstest.MapFile{Data: []byte("texto en castellano")},
		"exercises/ft_strlen/reference.c": &fstest.MapFile{Data: []byte("int main(void){return 0;}")},
		"exercises/ft_strlen/cases.txt":   &fstest.MapFile{Data: []byte("{\"args\": []}\n")},
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
	m, _ = m.Update(key("2"))
	m, _ = m.Update(key("enter"))
	m, _ = m.Update(key("x"))
	view := m.View()
	if strings.Contains(view, "castellano") || strings.Contains(view, "explicación") {
		t.Errorf("the exercise screen shows Spanish text:\n%s", view)
	}
}

func TestAnExamNoticeDoesNotFollowIntoPractice(t *testing.T) {
	m := newTestModel(t)
	exam, _ := m.Update(key("1"))
	passed, _ := exam.Update(gradedMsg{verdict: grader.Verdict{Status: grader.StatusOK, Summary: "OK"}})
	menu, _ := passed.Update(key("q"))
	list, _ := menu.Update(key("2"))
	practice, _ := list.Update(key("enter"))
	if strings.Contains(practice.View(), "last graded") {
		t.Errorf("the practice screen shows the exam's notice:\n%s", practice.View())
	}
}

func TestPracticeListIsSortedByLevel(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(key("2"))
	view := next.View()
	// testCatalog's names sort alphabetically as epur_str, ft_atoi,
	// ft_split, ft_strlen; by level they are ft_strlen(1), ft_atoi(2),
	// epur_str(3), ft_split(4).
	order := []string{"ft_strlen", "ft_atoi", "epur_str", "ft_split"}
	last := -1
	for _, name := range order {
		i := strings.Index(view, name)
		if i < last {
			t.Fatalf("%s is out of level order:\n%s", name, view)
		}
		last = i
	}
}

func TestTheExerciseScreenShowsTheAbsolutePathOfTheFile(t *testing.T) {
	root := t.TempDir()
	state, _ := store.Load(t.TempDir())
	m := New(Deps{
		Catalog: testCatalog(t), Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: t.TempDir(), Root: root,
	})
	exam, _ := m.Update(key("1"))
	want := filepath.Join(root, "rendu", exam.current.Name, exam.current.ExpectedFile)
	if !strings.Contains(exam.View(), want) {
		t.Errorf("the exercise screen does not show %s:\n%s", want, exam.View())
	}
}

func TestAKOShowsWhichFileWasGraded(t *testing.T) {
	m := newTestModel(t)
	list, _ := m.Update(key("2"))
	ex, _ := list.Update(key("enter"))
	ko, _ := ex.Update(gradedMsg{verdict: grader.Verdict{Status: grader.StatusWrongOutput, Summary: "wrong output"}})
	if !strings.Contains(ko.View(), "graded "+ko.srcPath) {
		t.Errorf("a KO does not say which file was graded:\n%s", ko.View())
	}
}

func TestGradingAnUntouchedStubWarnsInsteadOfGrading(t *testing.T) {
	m := newTestModel(t)
	exam, _ := m.Update(key("1"))
	next, cmd := exam.Update(key("g"))
	if cmd != nil || next.grading {
		t.Fatal("an untouched starter file was sent to the grader")
	}
	if len(next.exam.Attempts()) != 0 {
		t.Error("an untouched starter file cost an exam attempt")
	}
	if !strings.Contains(next.View(), "untouched") {
		t.Errorf("no warning about the untouched file:\n%s", next.View())
	}
}

func TestAResumedExamKeepsTheWorkspaceItStartedIn(t *testing.T) {
	pools := map[int][]string{1: {"ft_strlen"}, 2: {"ft_atoi"}, 3: {"epur_str"}, 4: {"ft_split"}}
	exam, err := session.NewExam(store.DefaultConfig(), pools, rand.New(rand.NewSource(1)), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	started := t.TempDir()
	exam.Root = started
	state, _ := store.Load(t.TempDir())
	m := New(Deps{
		Catalog: testCatalog(t), Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: t.TempDir(), Root: t.TempDir(), ResumeExam: exam,
	})
	resumed, _ := m.Update(key("c"))
	if !strings.HasPrefix(resumed.srcPath, started) {
		t.Errorf("resumed exam uses %s, want a file under %s where it started", resumed.srcPath, started)
	}
	if !strings.Contains(resumed.View(), started) {
		t.Errorf("the screen does not say where the exam's files are:\n%s", resumed.View())
	}
}

func TestTheMenuWarnsWhenLaunchedFromADifferentFolder(t *testing.T) {
	state, _ := store.Load(t.TempDir())
	state.Workspace = "/somewhere/else"
	m := New(Deps{
		Catalog: testCatalog(t), Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: t.TempDir(), Root: t.TempDir(),
	})
	if !strings.Contains(m.View(), "/somewhere/else") {
		t.Errorf("the menu does not mention the previous rendu/ folder:\n%s", m.View())
	}
}

func TestAnExamKONamesTheFileThatWasGradedNotTheNextOne(t *testing.T) {
	// Two level 1 exercises, so a failure must redraw the other one.
	fsys := fstest.MapFS{}
	for name, level := range map[string]int{"aaa": 1, "bbb": 1, "ft_atoi": 2, "epur_str": 3, "ft_split": 4} {
		dir := "exercises/" + name + "/"
		fsys[dir+"meta.yaml"] = &fstest.MapFile{Data: []byte(fmt.Sprintf(
			"name: %s\nlevel: %d\nkind: program\nexpected_file: %s.c\nallowed_functions: []\nprototype: \"\"\n", name, level, name))}
		fsys[dir+"subject.md"] = &fstest.MapFile{Data: []byte("subject for " + name)}
		fsys[dir+"reference.c"] = &fstest.MapFile{Data: []byte("int main(void){return 0;}")}
		fsys[dir+"cases.txt"] = &fstest.MapFile{Data: []byte("{\"args\": []}\n")}
	}
	c, err := catalog.Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	state, _ := store.Load(t.TempDir())
	m := New(Deps{
		Catalog: c, Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: t.TempDir(), Root: t.TempDir(),
	})
	exam, _ := m.Update(key("1"))
	graded := exam.srcPath
	next, _ := exam.Update(gradedMsg{verdict: grader.Verdict{Status: grader.StatusWrongOutput, Summary: "wrong output"}})
	if next.srcPath == graded {
		t.Fatal("the failure did not redraw a different exercise; the test proves nothing")
	}
	if !strings.Contains(next.View(), "graded "+graded) {
		t.Errorf("the KO does not name %s, the file that was graded:\n%s", graded, next.View())
	}
}

// The folder must be on disk as soon as rendu/ is used there, not only when
// the program quits cleanly: a closed terminal would otherwise lose it.
func TestOpeningAnExerciseRecordsTheWorkspaceOnDisk(t *testing.T) {
	stateDir, root := t.TempDir(), t.TempDir()
	state, _ := store.Load(stateDir)
	m := New(Deps{
		Catalog: testCatalog(t), Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: stateDir, Root: root,
	})
	list, _ := m.Update(key("2"))
	list.Update(key("enter"))
	saved, err := store.Load(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Workspace != root {
		t.Errorf("saved Workspace = %q, want %q", saved.Workspace, root)
	}
}

// modelWithState builds a model over state saved in stateDir, so a test can
// read back what the interface wrote to disk.
func modelWithState(t *testing.T, state *store.State, stateDir string) Model {
	t.Helper()
	return New(Deps{
		Catalog: testCatalog(t), Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: stateDir, Root: t.TempDir(),
	})
}

func TestMarkingInThePracticeListTogglesAndSaves(t *testing.T) {
	stateDir := t.TempDir()
	state, _ := store.Load(stateDir)
	list, _ := modelWithState(t, state, stateDir).Update(key("2"))

	marked, _ := list.Update(key("m"))
	if !marked.deps.State.IsUnsure("ft_strlen") {
		t.Fatalf("m on the first row did not mark ft_strlen")
	}
	if !strings.Contains(marked.View(), "?") {
		t.Errorf("the practice list does not show the mark:\n%s", marked.View())
	}
	saved, _ := store.Load(stateDir)
	if !saved.IsUnsure("ft_strlen") {
		t.Errorf("the mark was not saved to disk")
	}

	unmarked, _ := marked.Update(key("m"))
	if unmarked.deps.State.IsUnsure("ft_strlen") {
		t.Errorf("a second m did not unmark ft_strlen")
	}
}

func TestPracticeFilterShowsOnlyMarkedExercises(t *testing.T) {
	state, _ := store.Load(t.TempDir())
	state.ToggleUnsure("epur_str")
	list, _ := modelWithState(t, state, t.TempDir()).Update(key("2"))

	filtered, _ := list.Update(key("f"))
	if len(filtered.practice) != 1 || filtered.practice[0].Name != "epur_str" {
		t.Fatalf("filtered list = %v, want only epur_str", names(filtered.practice))
	}
	if strings.Contains(filtered.View(), "ft_strlen") {
		t.Errorf("an unmarked exercise is still listed:\n%s", filtered.View())
	}

	all, _ := filtered.Update(key("f"))
	if len(all.practice) != 4 {
		t.Errorf("after a second f the list has %d exercises, want all 4", len(all.practice))
	}
}

func TestAnEmptyFilterSaysHowToMark(t *testing.T) {
	list, _ := newTestModel(t).Update(key("2"))
	filtered, _ := list.Update(key("f"))
	if len(filtered.practice) != 0 {
		t.Fatalf("filtered list = %v, want empty", names(filtered.practice))
	}
	if !strings.Contains(filtered.View(), "no exercises marked unsure") {
		t.Errorf("an empty filter does not explain itself:\n%s", filtered.View())
	}
	// Keys that act on a row must not panic on an empty list.
	for _, k := range []string{"m", "enter", "j", "k", "u"} {
		filtered.Update(key(k))
	}
}

func TestUnmarkingInTheFilteredListDropsTheRowAndKeepsTheCursorInRange(t *testing.T) {
	state, _ := store.Load(t.TempDir())
	state.ToggleUnsure("ft_strlen")
	state.ToggleUnsure("ft_split")
	list, _ := modelWithState(t, state, t.TempDir()).Update(key("2"))
	filtered, _ := list.Update(key("f"))
	last, _ := filtered.Update(tea.KeyMsg{Type: tea.KeyDown})
	if last.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", last.cursor)
	}

	after, _ := last.Update(key("m"))
	if len(after.practice) != 1 || after.practice[0].Name != "ft_strlen" {
		t.Fatalf("filtered list after unmarking = %v, want only ft_strlen", names(after.practice))
	}
	if after.cursor != 0 {
		t.Errorf("cursor = %d after its row vanished, want 0", after.cursor)
	}
}

func TestDrillUnsureOpensTheMarkedOneAttemptedLongestAgo(t *testing.T) {
	state, _ := store.Load(t.TempDir())
	state.Record("ft_atoi", "OK", true)
	state.Exercises["ft_atoi"].LastAt = time.Now().Add(-48 * time.Hour)
	state.Record("ft_split", "OK", true)
	state.ToggleUnsure("ft_atoi")
	state.ToggleUnsure("ft_split")
	list, _ := modelWithState(t, state, t.TempDir()).Update(key("2"))

	drill, _ := list.Update(key("u"))
	if drill.screen != screenExercise || drill.current.Name != "ft_atoi" {
		t.Errorf("u opened %q on screen %d, want ft_atoi on the exercise screen", drill.current.Name, drill.screen)
	}
}

func TestDrillUnsureWithNothingMarkedSaysSo(t *testing.T) {
	list, _ := newTestModel(t).Update(key("2"))
	drill, _ := list.Update(key("u"))
	if drill.screen != screenPractice {
		t.Fatalf("u with nothing marked left the list for screen %d", drill.screen)
	}
	if !strings.Contains(drill.View(), "nothing is marked unsure") {
		t.Errorf("u with nothing marked gave no explanation:\n%s", drill.View())
	}
}

func TestMarkingOnThePracticeExerciseScreen(t *testing.T) {
	stateDir := t.TempDir()
	state, _ := store.Load(stateDir)
	list, _ := modelWithState(t, state, stateDir).Update(key("2"))
	ex, _ := list.Update(key("enter"))
	if !strings.Contains(ex.View(), "m") || !strings.Contains(ex.View(), "mark") {
		t.Errorf("the exercise screen does not offer m to mark:\n%s", ex.View())
	}

	marked, _ := ex.Update(key("m"))
	if !marked.deps.State.IsUnsure("ft_strlen") {
		t.Fatalf("m on the exercise screen did not mark ft_strlen")
	}
	if !strings.Contains(marked.View(), "marked unsure") {
		t.Errorf("the exercise screen does not show the mark:\n%s", marked.View())
	}
	if saved, _ := store.Load(stateDir); !saved.IsUnsure("ft_strlen") {
		t.Errorf("the mark was not saved to disk")
	}
}

func TestMarkingDuringAnExamLeavesTheExamAlone(t *testing.T) {
	m := newTestModel(t)
	exam, _ := m.Update(key("1"))
	name, level, points := exam.exam.Current(), exam.exam.Level(), exam.exam.Points()

	marked, _ := exam.Update(key("m"))
	if marked.screen != screenExam {
		t.Fatalf("m during an exam moved to screen %d", marked.screen)
	}
	if !marked.deps.State.IsUnsure(name) {
		t.Errorf("m during an exam did not mark %s", name)
	}
	if marked.exam.Current() != name || marked.exam.Level() != level || marked.exam.Points() != points {
		t.Errorf("marking changed the exam: now %s level %d %d points, was %s level %d %d points",
			marked.exam.Current(), marked.exam.Level(), marked.exam.Points(), name, level, points)
	}
	if !strings.Contains(marked.View(), "marked unsure") {
		t.Errorf("the exam screen does not show the mark:\n%s", marked.View())
	}
}

func TestLeavingAnExerciseRefreshesTheFilteredList(t *testing.T) {
	state, _ := store.Load(t.TempDir())
	state.ToggleUnsure("ft_strlen")
	list, _ := modelWithState(t, state, t.TempDir()).Update(key("2"))
	filtered, _ := list.Update(key("f"))
	ex, _ := filtered.Update(key("enter"))
	unmarked, _ := ex.Update(key("m"))
	back, _ := unmarked.Update(key("q"))
	if len(back.practice) != 0 {
		t.Errorf("filtered list after unmarking inside the exercise = %v, want empty", names(back.practice))
	}
}

func names(exs []catalog.Exercise) []string {
	out := make([]string, 0, len(exs))
	for _, ex := range exs {
		out = append(out, ex.Name)
	}
	return out
}

func TestAPracticeListErrorClearsOnTheNextKey(t *testing.T) {
	list, _ := newTestModel(t).Update(key("2"))
	list.err = errors.New("saving state: disk full")
	if !strings.Contains(list.View(), "disk full") {
		t.Fatalf("the practice list does not show an error:\n%s", list.View())
	}
	next, _ := list.Update(tea.KeyMsg{Type: tea.KeyDown})
	if strings.Contains(next.View(), "disk full") {
		t.Errorf("the error is still shown after another key:\n%s", next.View())
	}
}
