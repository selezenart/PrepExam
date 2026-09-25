package catalog

// Kind says how an exercise is built and run.
type Kind string

const (
	// KindFunction means the candidate writes a function with no main. The
	// project supplies a driver that calls it and prints the result.
	KindFunction Kind = "function"
	// KindProgram means the candidate writes a whole program, main included,
	// which is run directly on the case arguments.
	KindProgram Kind = "program"
)

// Case is one set of inputs an exercise is run against.
type Case struct {
	Args  []string `json:"args"`
	Stdin string   `json:"stdin,omitempty"`
}

// Meta is the hand-reviewed metadata stored as meta.yaml beside each exercise.
type Meta struct {
	Name             string   `yaml:"name"`
	Level            int      `yaml:"level"`
	Kind             Kind     `yaml:"kind"`
	ExpectedFile     string   `yaml:"expected_file"`
	AllowedFunctions []string `yaml:"allowed_functions"`
	Prototype        string   `yaml:"prototype"`
	Header           string   `yaml:"header,omitempty"`
	TimeoutMS        int      `yaml:"timeout_ms,omitempty"`
}

// Exercise is everything needed to present and grade one exercise.
type Exercise struct {
	Meta
	Subject       string // the upstream subject, shown to the candidate
	Reference     string // the oracle solution
	Driver        string // test main; empty when Kind is KindProgram
	HeaderContent string // contents of Header; empty when Header is empty
	Cases         []Case
}

// Gradable reports whether this exercise has the driver and cases needed to
// grade an answer. The importer generates every exercise's subject and
// reference solution, but drivers and cases are written by hand afterwards,
// so an exercise can legitimately exist without yet being gradable.
func (e Exercise) Gradable() bool {
	if len(e.Cases) == 0 {
		return false
	}
	return e.Kind != KindFunction || e.Driver != ""
}

// Timeout is how long one run of this exercise may take, in milliseconds.
func (e Exercise) Timeout() int {
	if e.TimeoutMS > 0 {
		return e.TimeoutMS
	}
	return 5000
}
