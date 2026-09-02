package grader

import "runtime"

// compilerEmitted lists symbols a compiler synthesises from code that never
// names them: memcpy and memset for aggregate assignment and large
// initialisers, the stack-protector hooks, and the runtime entry points a
// linked object refers to.
//
// Reporting one of these as a forbidden call would fail a candidate for
// something they did not write, so they are subtracted before the check. The
// lists differ by platform because clang on macOS and gcc on Linux emit
// different names.
var compilerEmitted = func() map[string]bool {
	shared := []string{
		"memcpy", "memmove", "memset", "bcopy",
		"__stack_chk_fail", "__stack_chk_guard",
	}
	platform := map[string][]string{
		"darwin": {
			"___stack_chk_fail", "___stack_chk_guard",
			"dyld_stub_binder", "__Unwind_Resume",
		},
		"linux": {
			"__stack_chk_fail_local", "_GLOBAL_OFFSET_TABLE_",
			"__gmon_start__", "_Unwind_Resume",
		},
	}
	set := make(map[string]bool)
	for _, s := range append(shared, platform[runtime.GOOS]...) {
		set[s] = true
	}
	return set
}()
