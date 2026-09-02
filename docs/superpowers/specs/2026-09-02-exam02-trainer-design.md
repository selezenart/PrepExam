# Exam Rank 02 Trainer — Design

**Date:** 2026-09-02
**Status:** Approved, ready for implementation planning

## Purpose

A terminal application for practising the 42 School Exam Rank 02. It presents
exercise subjects, gives the user a workspace to solve them in, and grades the
result automatically — including the allowed-functions check that the real
Moulinette enforces and that most informal trainers omit.

Two modes: a timed exam simulation and an untimed free-practice mode.

Ships as a single static binary for macOS and Linux, cross-compiled from a
Linux host.

## Source material

Exercise data comes from
[alexhiguera/Exam_Rank_02_42_School](https://github.com/alexhiguera/Exam_Rank_02_42_School)
(MIT, Copyright (c) 2026 Alex Higuera), vendored at
`third_party/exam_rank_02/`. The upstream `LICENSE` is preserved there, and the
application's own README and about screen credit the source.

The repository provides, per exercise: the original subject (`README.md`), a
Spanish explainer (`spanish.md`), a reference solution (`<name>.c`), and for
five exercises a header file.

**56 exercises:** Level 1 — 12, Level 2 — 19, Level 3 — 15, Level 4 — 10.

The repository provides no test cases. Test drivers and input vectors are
authored as part of this project.

## Success criteria

1. A user on macOS or Linux downloads one binary, runs it, and can attempt any
   of the 56 exercises with no further installation beyond a working `cc`.
2. Grading a correct solution returns OK; grading a solution that calls a
   disallowed function returns KO naming that function.
3. All 56 reference solutions pass their own graders (integration test).
4. A single-character mutation of any reference solution fails its grader
   (negative test).

## Architecture

```
cmd/exam02/main.go          entry point, flag parsing, cc preflight
internal/ui/                Bubble Tea models: menu, exam, practice, result, about
internal/catalog/           embedded exercise data, meta parsing, lookup
internal/session/           exam state machine, timer, scoring, persistence
internal/grader/            compile → symbols → link → run → diff
internal/sandbox/           process execution with timeout and group kill
internal/config/            defaults, config file load
tools/import/               one-shot importer: third_party → data/exercises
data/exercises/<name>/      generated + hand-authored exercise data (embedded)
```

Each package has one responsibility and a narrow interface. `grader` knows
nothing about the TUI; `ui` calls `grader.Grade(exercise, path) → Verdict` and
renders whatever comes back. `session` owns exam rules and never runs a
compiler. This keeps the compiler-facing code testable without a terminal and
the exam rules testable without a compiler.

### Exercise data format

```
data/exercises/ft_split/
  subject.md      copied from upstream README.md
  spanish.md      copied verbatim, shown in practice mode only
  reference.c     the oracle
  meta.yaml       metadata, see below
  driver.c        test main; present for kind: function only
  cases.txt       input vectors
  header.h        only where upstream ships one (5 exercises)
```

`meta.yaml`:

```yaml
name: ft_split
level: 4
kind: function            # function | program
expected_file: ft_split.c
allowed_functions: [malloc]
prototype: "char **ft_split(char *str);"
header: ft_split.h        # optional, omitted when absent
timeout_ms: 5000          # optional, defaults to 5000
```

`name`, `expected_file`, `allowed_functions` and `prototype` are parsed from
the upstream subject by `tools/import`. `level` comes from the directory.

`kind` is **hand-authored**. A phrase heuristic ("Your function must be declared
as follows" vs "Write a program") classifies only 43 of 56 subjects; the
remaining 13 are ambiguous. The importer writes its best guess as a starting
value, and every entry is reviewed by hand. This costs nothing extra because
each exercise already needs a hand-written driver and case set.

`allowed_functions` parsing handles the observed upstream spellings: a
comma-separated list, the literal `None`, the literal `-`, and an empty value.
The last three all mean the empty set.

### Grading pipeline

`grader.Grade` runs these stages in order and stops at the first failure.

**1. Collect.** Read `rendu/<name>/<expected_file>` relative to the working
directory. Missing file → KO, "expected file not found".

**2. Compile.** `cc -Wall -Wextra -Werror -c <user file> -o <tmp>/user.o`.
Non-zero exit → KO, "compile error", with the compiler's stderr shown verbatim.
This matches the real exam, where a warning is a failure.

**3. Forbidden functions.** `nm -u <tmp>/user.o` lists undefined symbols. From
that set subtract:

- the exercise's `allowed_functions`
- the exercise's own entry point (a function exercise may call itself)
- a per-platform whitelist of compiler-emitted builtins

Anything left → KO, "forbidden function: <name>".

The builtin whitelist exists because compilers synthesise calls the user never
wrote: `memcpy` and `memset` for aggregate assignment and large initialisers,
`__stack_chk_fail` and `__stack_chk_guard` from stack protection. The macOS and
Linux lists differ and are maintained separately. This whitelist is the single
most likely source of false-positive failures, so it is unit-tested against
compiled fixtures on both platforms.

**4. Link.**

- `kind: function` — `cc user.o driver.o → user_bin` and
  `cc reference.o driver.o → ref_bin`. The driver supplies `main`, builds
  inputs, calls the exercise function, and prints the result in a form that
  makes a wrong answer visible (for `ft_split`, every word on its own line
  and a terminator marker; for `ft_range`, every integer).
- `kind: program` — `cc user.c → user_bin` and `cc reference.c → ref_bin`. The
  user's file contains `main` itself, so no driver is linked.

**5. Run.** For each case in `cases.txt`, execute `user_bin` and `ref_bin` with
identical argv and stdin, under the exercise's timeout. Capture stdout, stderr,
exit status, and terminating signal.

**6. Diff.** Compare stdout, then exit status, then signal. First mismatch →
KO, showing the case, the expected output, and the actual output. A terminating
signal is reported in the user's terms: SIGSEGV as "segfault", SIGABRT as
"abort", timeout as "timed out after 5s". All cases matching → OK.

Because the reference is the oracle, a bug shared by the reference and the user
will not be caught. This is accepted: the tool trains against the reference's
behaviour, which is the same standard the upstream repository already sets.

### Test cases

`cases.txt` holds one case per line. Each line is an argv vector in shell-like
quoting; stdin is not used by any upstream exercise, but the format reserves a
`stdin:` prefix so the runner does not need reworking if that changes.

Cases are hand-curated per exercise and must cover, where meaningful: the empty
input, a single element, the ordinary case, the boundary case named in the
subject, and any case the subject explicitly calls out. Cases must avoid input
whose correct behaviour is undefined by the subject — integer overflow in
`ft_atoi`, for example — because the reference's behaviour there is arbitrary
rather than authoritative.

### Sandbox

Child processes run with a timeout enforced in Go, never via `timeout(1)`,
which macOS does not ship. On timeout the child's whole process group is
killed, so a fork bomb or a stuck grandchild cannot outlive the attempt.
Working directory is a per-attempt temporary directory, removed afterwards.

Resource limits beyond the timeout are deliberately not imposed: `RLIMIT_AS`
behaves differently enough across macOS and Linux that the portability cost
exceeds the benefit, and the timeout already bounds runaway allocation in
practice.

## Modes

### Exam

- Countdown timer over the whole session.
- Starts at Level 1 with a random exercise from that level.
- OK → advance a level, draw a new random exercise.
- KO → draw a new random exercise from the same level. Attempts are unlimited
  within the time limit.
- No solution reveal, no Spanish explainer.
- Session ends on time expiry, on completing Level 4, or on quit. The result
  screen shows the level reached, points, elapsed time, and a per-exercise
  history.

**Defaults are assumptions, not verified exam rules.** The real Moulinette's
time limit, points table, and retry-penalty behaviour are not documented in a
source this project has verified, and inventing precise numbers and presenting
them as authentic would be worse than stating the gap. Defaults: three hours,
one passing exercise required to clear a level, no cooldown after a
failure, points 1/2/3/4 by level.
All live in `config.yaml` next to the state file and are changeable in one
line. If the real values are supplied later, they become the defaults.

### Practice

- Untimed.
- Browse all 56 by level, pick by name, pick at random, or drill the weakest
  (highest failure count, ties broken by least recently attempted).
- Unlimited grading attempts.
- `[r]` reveals the reference solution, `[x]` reveals the Spanish explainer.
- Per-exercise statistics: attempts, passes, last verdict, best time.

## Workspace and editing

The workspace mirrors the real exam: `./rendu/<name>/<expected_file>` under the
current working directory. On starting an exercise the file is created if
absent, seeded with the 42 header comment and the prototype as a stub, and the
exercise's `header.h` is copied alongside it when one exists. An existing file
is never overwritten.

`[e]` suspends the TUI, runs `$EDITOR` (default `vim`) on the file, and resumes
on exit.

## Persistence

State lives at `os.UserConfigDir()/exam02/state.json` — `~/.config/exam02` on
Linux, `~/Library/Application Support/exam02` on macOS. It holds per-exercise
statistics and any in-progress exam session, so a crash or an accidental quit
does not destroy a run. Writes are atomic: write to a temporary file in the same
directory, then rename.

`config.yaml` sits beside it and holds the exam-rule defaults above.

## Testing

**Unit.** Subject parser (name, expected file, allowed functions, prototype,
across every upstream spelling), symbol normaliser, builtin whitelist against
compiled fixtures, case-file parser, diff formatter, exam state machine
(advance, fail, timer expiry, resume).

**Integration — the load-bearing test.** For all 56 exercises, copy the
reference solution into a temporary workspace and grade it. Every one must
return OK. This validates every driver, every case file, and the whole
pipeline in one run, and it fails loudly the moment a driver is wrong.

**Negative.** For each exercise, apply a mechanical mutation to the reference
(flip a comparison operator, shift a loop bound) and grade it. Every one must
return KO. Without this, a driver that prints nothing would pass the
integration test while testing nothing.

**Forbidden-function.** Fixtures that call `printf` where only `write` is
allowed must be rejected, naming `printf`. Fixtures that trigger a
compiler-emitted `memcpy` must be accepted.

## Cross-platform handling

| Concern | Handling |
| --- | --- |
| `nm -u` prints `_printf` on macOS, `printf` on Linux | Strip a single leading underscore on darwin before comparison |
| Compiler-emitted builtins differ between clang and gcc | Separate whitelists per platform, tested against fixtures |
| `timeout(1)` absent on macOS | Timeout implemented in Go |
| `cc` may be missing | Preflight check at startup; on macOS advise `xcode-select --install`, on Linux advise the distribution's build-essential equivalent |
| Config directory differs | `os.UserConfigDir()` |

## Distribution

`make dist` cross-compiles from Linux to `darwin/arm64`, `darwin/amd64`,
`linux/amd64`, and `linux/arm64`, producing `dist/exam02_<os>_<arch>` plus a
`SHA256SUMS` file. Exercise data is embedded with `go:embed`, so each binary is
self-contained. No macOS host is required to produce the macOS builds.

macOS binaries are unsigned. Gatekeeper will quarantine a downloaded unsigned
binary, so the README documents `xattr -d com.apple.quarantine`. Signing and
notarisation need an Apple Developer account and are out of scope.

## Out of scope

Leak detection, norminette checking, a spaced-repetition algorithm beyond
"weakest first", translation of the interface into Spanish (the upstream
Spanish explainers are still shown verbatim), a Homebrew tap, and any
multi-user or networked feature.

## Build order

1. Engine and Level 1 only: catalog, grader, sandbox, minimal TUI, plus drivers
   and cases for all 12 Level 1 exercises, with the integration and negative
   tests green.
2. Exam and practice modes, persistence, editor integration.
3. Level 2 drivers and cases (19).
4. Level 3 drivers and cases (15), including the first linked-list exercise.
5. Level 4 drivers and cases (10), including `flood_fill` and the remaining
   linked-list exercises.
6. Release pipeline and documentation.

Step 1 is deliberately a vertical slice: it proves the entire pipeline against
real code before the remaining 44 drivers are written.
