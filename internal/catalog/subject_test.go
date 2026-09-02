package catalog

import (
	"reflect"
	"testing"
)

const strlenSubject = "## Subject\n\n```BASH\n" +
	"Assignment name  : ft_strlen\n" +
	"Expected files   : ft_strlen.c\n" +
	"Allowed functions: \n" +
	"--------------------------------------------------------------------------------\n\n" +
	"Write a function that returns the length of a string.\n\n" +
	"Your function must be declared as follows:\n\n" +
	"int\tft_strlen(char *str);\n```\n"

func TestParseSubjectReadsHeaderFields(t *testing.T) {
	got, err := ParseSubject(strlenSubject)
	if err != nil {
		t.Fatalf("ParseSubject() error = %v", err)
	}
	if got.Name != "ft_strlen" {
		t.Errorf("Name = %q, want %q", got.Name, "ft_strlen")
	}
	if got.ExpectedFile != "ft_strlen.c" {
		t.Errorf("ExpectedFile = %q, want %q", got.ExpectedFile, "ft_strlen.c")
	}
	if len(got.AllowedFunctions) != 0 {
		t.Errorf("AllowedFunctions = %v, want empty", got.AllowedFunctions)
	}
}

func TestParseSubjectAllowedFunctions(t *testing.T) {
	tests := []struct {
		name  string
		field string
		want  []string
	}{
		{"empty means none", "", nil},
		{"literal None", "None", nil},
		{"literal dash", "-", nil},
		{"single", "malloc", []string{"malloc"}},
		{"list", "write, malloc, free", []string{"write", "malloc", "free"}},
		{"ragged spacing", "  write ,malloc  ", []string{"write", "malloc"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := "Assignment name  : x\nExpected files   : x.c\nAllowed functions: " + tt.field + "\n"
			got, err := ParseSubject(md)
			if err != nil {
				t.Fatalf("ParseSubject() error = %v", err)
			}
			if !reflect.DeepEqual(got.AllowedFunctions, tt.want) {
				t.Errorf("AllowedFunctions = %#v, want %#v", got.AllowedFunctions, tt.want)
			}
		})
	}
}

func TestParseSubjectMultipleExpectedFilesTakesTheC(t *testing.T) {
	md := "Assignment name  : ft_list_size\nExpected files   : ft_list_size.c, ft_list.h\nAllowed functions: \n"
	got, err := ParseSubject(md)
	if err != nil {
		t.Fatalf("ParseSubject() error = %v", err)
	}
	if got.ExpectedFile != "ft_list_size.c" {
		t.Errorf("ExpectedFile = %q, want %q", got.ExpectedFile, "ft_list_size.c")
	}
}

func TestParseSubjectGlobExpectedFilesIsUnresolved(t *testing.T) {
	// flood_fill and sort_list write "*.c, *.h". A glob cannot be resolved
	// automatically, so the parser reports no expected file and meta.yaml
	// supplies one by hand.
	md := "Assignment name  : flood_fill\nExpected files   : *.c, *.h\nAllowed functions: \n"
	got, err := ParseSubject(md)
	if err != nil {
		t.Fatalf("ParseSubject() error = %v", err)
	}
	if got.ExpectedFile != "" {
		t.Errorf("ExpectedFile = %q, want empty for a glob", got.ExpectedFile)
	}
}

func TestParseSubjectWithoutAssignmentNameIsAnError(t *testing.T) {
	if _, err := ParseSubject("just prose, no header\n"); err == nil {
		t.Fatal("ParseSubject() error = nil, want an error")
	}
}
