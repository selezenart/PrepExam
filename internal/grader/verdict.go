package grader

// Status is the outcome of a grading attempt.
type Status string

const (
	StatusOK             Status = "OK"
	StatusMissingFile    Status = "missing file"
	StatusCompileError   Status = "compile error"
	StatusForbidden      Status = "forbidden function"
	StatusUnexpectedMain Status = "unexpected main"
	StatusWrongOutput    Status = "wrong output"
	StatusCrash          Status = "crash"
	StatusTimeout        Status = "timeout"
)

// Verdict is the result of grading one attempt.
type Verdict struct {
	Status  Status
	Summary string // one line, always set
	Detail  string // compiler output or an input/output comparison; may be empty
}

// Passed reports whether the attempt would score points.
func (v Verdict) Passed() bool { return v.Status == StatusOK }
