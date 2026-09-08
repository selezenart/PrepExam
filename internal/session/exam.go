// Package session holds the rules of an exam run: which exercise is current,
// what a pass or a fail does, and when the run is over. It touches neither the
// compiler nor the terminal, so every rule here is testable on its own.
package session

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"exam02/internal/store"
)

// levels is how many levels an exam covers. One exercise is drawn from each,
// so four passes at 25 points each is the 100-point pass mark.
const levels = 4

// Attempt is one graded submission during an exam.
type Attempt struct {
	Exercise string    `json:"exercise"`
	Level    int       `json:"level"`
	Passed   bool      `json:"passed"`
	At       time.Time `json:"at"`
}

// Exam is one run: four exercises, one drawn from each level, against a clock.
type Exam struct {
	StartedAt time.Time     `json:"started_at"`
	Duration  time.Duration `json:"duration"`
	PerLevel  int           `json:"per_level"`
	PassMark  int           `json:"pass_mark"`

	CurrentLevel    int       `json:"current_level"`
	CurrentExercise string    `json:"current_exercise"`
	Cleared         int       `json:"cleared"`
	History         []Attempt `json:"history"`

	pools        map[int][]string
	rng          *rand.Rand
	substitution *Substitution
}

// Substitution describes an exercise Decode had to swap in because the one
// saved in state no longer exists in its level's pool — for example, the
// catalog changed between saves. It is nil on a normal resume where nothing
// had to change.
type Substitution struct {
	From string
	To   string
}

// Substitution reports the swap Decode made to resume this exam, or nil if
// none was needed.
func (e *Exam) Substitution() *Substitution { return e.substitution }

// NewExam starts a run, drawing the first exercise from the Level 1 pool.
//
// pools maps a level to the names of its exercises. Every level must have at
// least one: an exam that cannot draw an exercise for a level could never be
// passed, and failing at the start says so more clearly than failing later.
func NewExam(cfg store.Config, pools map[int][]string, rng *rand.Rand, now time.Time) (*Exam, error) {
	if err := validatePools(pools, 1); err != nil {
		return nil, err
	}
	e := &Exam{
		StartedAt:    now,
		Duration:     cfg.ExamDuration,
		PerLevel:     cfg.PointsPerExercise,
		PassMark:     cfg.PassMark,
		CurrentLevel: 1,
		pools:        clonePools(pools),
		rng:          rng,
	}
	e.draw()
	return e, nil
}

// Attach restores the pools and randomness after an exam has been decoded from
// saved state, which cannot carry either.
//
// pools is copied rather than aliased: a caller that goes on to mutate the
// map or its slices must not be able to reach back in and change what the
// exam draws from.
func (e *Exam) Attach(pools map[int][]string, rng *rand.Rand) {
	e.pools, e.rng = clonePools(pools), rng
}

// validatePools checks that every level from start through the last has at
// least one exercise to draw from. A level with none — including one not yet
// reached — could never be drawn from when the run gets there, and failing
// now says so far more clearly than stranding the run partway through.
func validatePools(pools map[int][]string, start int) error {
	for level := start; level <= levels; level++ {
		if len(pools[level]) == 0 {
			return fmt.Errorf("level %d has no exercises to draw from", level)
		}
	}
	return nil
}

// clonePools makes an independent copy of pools, so a caller mutating the map
// or its slices after handing them to an Exam cannot alter what it draws
// from.
func clonePools(pools map[int][]string) map[int][]string {
	out := make(map[int][]string, len(pools))
	for level, names := range pools {
		cp := make([]string, len(names))
		copy(cp, names)
		out[level] = cp
	}
	return out
}

// contains reports whether name appears in pool.
func contains(pool []string, name string) bool {
	for _, p := range pool {
		if p == name {
			return true
		}
	}
	return false
}

// draw picks a fresh exercise from the current level's pool, avoiding the one
// just attempted where the pool offers an alternative.
//
// This scans for candidates rather than resampling until a different name
// turns up: resampling would spin forever on a pool whose only distinct
// entry equals the one just attempted (e.g. duplicate names in the pool),
// which is exactly the wrong place to hang — mid-exam, against the clock.
func (e *Exam) draw() {
	pool := e.pools[e.CurrentLevel]
	if len(pool) == 0 {
		e.CurrentExercise = ""
		return
	}
	candidates := make([]string, 0, len(pool))
	for _, name := range pool {
		if name != e.CurrentExercise {
			candidates = append(candidates, name)
		}
	}
	if len(candidates) == 0 {
		// Every entry equals the one just attempted (a pool of one, or of
		// duplicates); there is nothing else to offer.
		candidates = pool
	}
	e.CurrentExercise = candidates[e.rng.Intn(len(candidates))]
}

// Record notes the outcome of grading the current exercise. Passing clears the
// level and draws from the next one; failing draws again from the same level,
// which costs time rather than points.
func (e *Exam) Record(passed bool, now time.Time) {
	if e.Over(now) {
		return
	}
	e.History = append(e.History, Attempt{
		Exercise: e.CurrentExercise,
		Level:    e.CurrentLevel,
		Passed:   passed,
		At:       now,
	})
	if !passed {
		e.draw()
		return
	}
	e.Cleared++
	if e.Cleared >= levels {
		e.CurrentExercise = ""
		return
	}
	e.CurrentLevel++
	e.draw()
}

// Current is the exercise being attempted, or "" once the exam is over.
func (e *Exam) Current() string { return e.CurrentExercise }

// Level is the level being attempted.
func (e *Exam) Level() int { return e.CurrentLevel }

// Points is the score so far.
func (e *Exam) Points() int { return e.Cleared * e.PerLevel }

// Passed reports whether the score has reached the pass mark.
func (e *Exam) Passed() bool { return e.Points() >= e.PassMark }

// Remaining is the time left, floored at zero.
func (e *Exam) Remaining(now time.Time) time.Duration {
	left := e.Duration - now.Sub(e.StartedAt)
	if left < 0 {
		return 0
	}
	return left
}

// Over reports whether the run has finished, by the clock or by clearing every
// level.
func (e *Exam) Over(now time.Time) bool {
	return e.Cleared >= levels || e.Remaining(now) == 0
}

// Attempts is the history of graded submissions, oldest first. It returns a
// copy: mutating the result must not reach back into the exam's own state.
func (e *Exam) Attempts() []Attempt {
	out := make([]Attempt, len(e.History))
	copy(out, e.History)
	return out
}

// Encode serialises an exam for storage. The draw pools and the random source
// are deliberately left out: pools are rebuilt from the catalog on load, so a
// saved exam cannot pin the application to a stale copy of the exercise list.
func Encode(e *Exam) (json.RawMessage, error) {
	raw, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("encoding the exam: %w", err)
	}
	return raw, nil
}

// Decode restores a saved exam and reattaches the pools and randomness that
// could not be serialised.
//
// The start time is restored as it was, so the clock carries on from where it
// stopped. Restarting it would turn quitting and reopening into a way to buy
// unlimited time.
//
// The freshly attached pools are re-validated against the exam being
// resumed, because the catalog — and so the pools rebuilt from it — can have
// changed since the exam was saved:
//
//   - If the current level, or one not yet reached, now has no exercises at
//     all, resuming can only strand the run later: Decode fails outright,
//     the same way NewExam refuses to start such a run in the first place.
//   - If the level's pool is otherwise healthy but no longer contains the
//     saved exercise (renamed or removed), the run is still completable, so
//     Decode redraws rather than erroring — discarding a run in progress
//     over one renamed exercise would be disproportionate. The swap is
//     recorded on Substitution, so a caller can tell the candidate their
//     exercise changed rather than silently handing them a different
//     problem than the one their in-progress work was written for.
//
// A finished exam (all four levels cleared) has nothing left to draw, so
// neither check applies to it.
func Decode(raw json.RawMessage, pools map[int][]string, rng *rand.Rand) (*Exam, error) {
	var e Exam
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil, fmt.Errorf("decoding the exam: %w", err)
	}
	if e.CurrentLevel < 1 || e.CurrentLevel > levels {
		return nil, fmt.Errorf("saved exam has an impossible level %d", e.CurrentLevel)
	}
	if e.CurrentExercise != "" {
		if err := validatePools(pools, e.CurrentLevel); err != nil {
			return nil, err
		}
	}
	e.Attach(pools, rng)
	if e.CurrentExercise != "" && !contains(e.pools[e.CurrentLevel], e.CurrentExercise) {
		from := e.CurrentExercise
		e.draw()
		e.substitution = &Substitution{From: from, To: e.CurrentExercise}
	}
	return &e, nil
}
