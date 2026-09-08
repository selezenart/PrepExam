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

	pools map[int][]string
	rng   *rand.Rand
}

// NewExam starts a run, drawing the first exercise from the Level 1 pool.
//
// pools maps a level to the names of its exercises. Every level must have at
// least one: an exam that cannot draw an exercise for a level could never be
// passed, and failing at the start says so more clearly than failing later.
func NewExam(cfg store.Config, pools map[int][]string, rng *rand.Rand, now time.Time) (*Exam, error) {
	for level := 1; level <= levels; level++ {
		if len(pools[level]) == 0 {
			return nil, fmt.Errorf("level %d has no exercises to draw from", level)
		}
	}
	e := &Exam{
		StartedAt:    now,
		Duration:     cfg.ExamDuration,
		PerLevel:     cfg.PointsPerExercise,
		PassMark:     cfg.PassMark,
		CurrentLevel: 1,
		pools:        pools,
		rng:          rng,
	}
	e.draw()
	return e, nil
}

// Attach restores the pools and randomness after an exam has been decoded from
// saved state, which cannot carry either.
func (e *Exam) Attach(pools map[int][]string, rng *rand.Rand) {
	e.pools, e.rng = pools, rng
}

// draw picks a fresh exercise from the current level's pool, avoiding the one
// just attempted where the pool is big enough to allow it.
func (e *Exam) draw() {
	pool := e.pools[e.CurrentLevel]
	if len(pool) == 0 {
		e.CurrentExercise = ""
		return
	}
	if len(pool) == 1 {
		e.CurrentExercise = pool[0]
		return
	}
	for {
		candidate := pool[e.rng.Intn(len(pool))]
		if candidate != e.CurrentExercise {
			e.CurrentExercise = candidate
			return
		}
	}
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

// Attempts is the history of graded submissions, oldest first.
func (e *Exam) Attempts() []Attempt { return e.History }

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
func Decode(raw json.RawMessage, pools map[int][]string, rng *rand.Rand) (*Exam, error) {
	var e Exam
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil, fmt.Errorf("decoding the exam: %w", err)
	}
	if e.CurrentLevel < 1 || e.CurrentLevel > levels {
		return nil, fmt.Errorf("saved exam has an impossible level %d", e.CurrentLevel)
	}
	e.Attach(pools, rng)
	return &e, nil
}
