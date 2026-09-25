# exam02

A terminal trainer for the 42 Common Core Exam Rank 02.

It shows you a subject, gives you a workspace, and grades what you write —
including the allowed-functions check that the real Moulinette enforces and
that most informal practice setups skip.

## Install

Download the binary for your platform, make it executable, and put it on your
PATH:

```bash
chmod +x exam02_darwin_arm64
mv exam02_darwin_arm64 /usr/local/bin/exam02
```

The macOS builds are unsigned, so Gatekeeper quarantines them on download.
Clear it once:

```bash
xattr -d com.apple.quarantine /usr/local/bin/exam02
```

You need a C toolchain — `cc` and `nm`. macOS: `xcode-select --install`.
Debian or Ubuntu: `sudo apt install build-essential`.

## Use

```bash
mkdir practice && cd practice
exam02
```

`rendu/` is created in the current directory, exactly as in the real exam.

- **Exam** — three hours, one exercise drawn from each of the four levels.
  25 points each, 100 to pass, so passing means clearing every level. Failing
  an exercise draws another from the same level: it costs you time, not points.
- **Practice** — any exercise, no clock, unlimited attempts, and the reference
  solution available with `r` when you want it.

Keys: `g` grade, `e` edit in `$EDITOR` (`vim` by default), `r` reveal the
solution in practice, `w` drill your weakest exercise, `q` back.

## How grading works

Your file is compiled with `cc -Wall -Wextra -Werror`, so a warning fails you,
as it does in the real exam. Then `nm` is used to check you have not called a
function the subject forbids. Then your code and the reference solution are run
on the same inputs and their output compared.

The reference solutions are the oracle, which is why no expected outputs are
written by hand — and it means a bug shared by your code and the reference will
not be caught. Treat this as practice, not as the Moulinette.

## Build

```bash
make test     # run the suite
make build    # build for this machine
make dist     # cross-compile all four platforms
```

The suite's most important test grades every reference solution against its own
grader. If a driver or a case file is wrong, that test fails.

## Content status

All 56 exercises across the four levels are gradable.

## Credit

Exercise subjects and the reference solutions come
from [alexhiguera/Exam_Rank_02_42_School](https://github.com/alexhiguera/Exam_Rank_02_42_School),
MIT licensed, Copyright (c) 2026 Alex Higuera. The upstream licence is kept at
`third_party/exam_rank_02/LICENSE`.

This is a practice tool. The real exam runs in a closed environment under the
Moulinette, and its rules are the ones that count.
