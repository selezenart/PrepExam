package grader

// compilerEmittedShared lists symbols, in normalized form, that a compiler
// synthesises from code that never names them, and that mean the same thing
// on every supported platform: memcpy and memset for aggregate assignment
// and large initialisers, the stack-protector hooks, and the C++/exception
// unwind-resume entry point.
//
// Reporting one of these as a forbidden call would fail a candidate for
// something they did not write, so they are subtracted before the check.
//
// Every entry here and in compilerEmittedPlatform is written in normalized
// form — the form normalizeSymbol produces after stripping darwin's ABI
// underscore — because Forbidden compares against already-normalized
// symbols. An entry written in a raw, unnormalized darwin form would never
// match what a real darwin build produces; see the darwin comments below
// for the reasoning per symbol.
var compilerEmittedShared = []string{
	"memcpy", "memmove", "memset", "bcopy",
	"__stack_chk_fail", "__stack_chk_guard",
	// _Unwind_Resume is the actual C identifier (one leading underscore) on
	// both platforms. Linux nm prints it unchanged. Darwin nm prints it with
	// one more, ABI-added underscore (__Unwind_Resume), which normalizeSymbol
	// strips back down to this same one-underscore name — so one shared
	// entry covers both platforms; it must not also appear, unnormalized, in
	// the darwin list below.
	"_Unwind_Resume",
}

// compilerEmittedPlatform lists symbols, in normalized form, that only one
// platform's toolchain or runtime emits.
var compilerEmittedPlatform = map[string][]string{
	"darwin": {
		// dyld_stub_binder is the lazy-binding stub every dynamically linked
		// Mach-O object references. Its C identifier has no leading
		// underscore; darwin nm shows it as _dyld_stub_binder (one ABI
		// underscore), which normalizeSymbol strips to this.
		//
		// __stack_chk_fail and __stack_chk_guard are NOT listed here: their C
		// identifiers already carry two leading underscores, so darwin nm
		// shows them as ___stack_chk_fail / ___stack_chk_guard (three:
		// two real, one ABI-added). normalizeSymbol strips exactly one,
		// leaving two — which is already the shared entry above. Listing the
		// three-underscore raw form here as well, as an earlier version of
		// this file did, added an entry Forbidden could never reach: nothing
		// it compares against ever has three leading underscores, since
		// normalizeSymbol always strips exactly one.
		"dyld_stub_binder",
	},
	"linux": {
		"__stack_chk_fail_local", "_GLOBAL_OFFSET_TABLE_",
		"__gmon_start__",
	},
}

// compilerEmittedFor returns the compiler-emitted symbol names for goos, in
// the normalized form parseNMOutput produces. It is a function of goos
// rather than a value computed once from runtime.GOOS so a test can check a
// specific platform's allowlist without depending on which platform the
// test binary happens to run on.
func compilerEmittedFor(goos string) map[string]bool {
	platform := compilerEmittedPlatform[goos]
	set := make(map[string]bool, len(compilerEmittedShared)+len(platform))
	for _, s := range compilerEmittedShared {
		set[s] = true
	}
	for _, s := range platform {
		set[s] = true
	}
	return set
}
