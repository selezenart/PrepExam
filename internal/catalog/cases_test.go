package catalog

import (
	"reflect"
	"testing"
)

func TestParseCases(t *testing.T) {
	in := `{"args": ["hello world"]}
{"args": [""]}
{"args": []}

{"args": ["a", "b"], "stdin": "x\n"}
`
	got, err := ParseCases(in)
	if err != nil {
		t.Fatalf("ParseCases() error = %v", err)
	}
	want := []Case{
		{Args: []string{"hello world"}},
		{Args: []string{""}},
		{Args: []string{}},
		{Args: []string{"a", "b"}, Stdin: "x\n"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseCases() = %#v, want %#v", got, want)
	}
}

func TestParseCasesDistinguishesEmptyArgFromNoArg(t *testing.T) {
	// Several exercises print different things for ./prog "" and ./prog.
	// This distinction is the whole reason the format is JSON.
	got, err := ParseCases("{\"args\": [\"\"]}\n{\"args\": []}\n")
	if err != nil {
		t.Fatalf("ParseCases() error = %v", err)
	}
	if len(got[0].Args) != 1 || got[0].Args[0] != "" {
		t.Errorf("first case Args = %#v, want one empty string", got[0].Args)
	}
	if len(got[1].Args) != 0 {
		t.Errorf("second case Args = %#v, want none", got[1].Args)
	}
}

func TestParseCasesRejectsMalformedLine(t *testing.T) {
	if _, err := ParseCases("not json\n"); err == nil {
		t.Fatal("ParseCases() error = nil, want an error")
	}
}

func TestParseCasesRejectsEmptyFile(t *testing.T) {
	// An exercise with no cases would silently pass everything.
	if _, err := ParseCases("\n\n"); err == nil {
		t.Fatal("ParseCases() error = nil, want an error for a caseless file")
	}
}
