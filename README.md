# Counter

A small command-line counter for Linux, macOS, and other Unix systems. Each
counter is a plain text file holding one integer, so counters survive reboots,
can be shared between scripts, and are safe to update from many processes at
once.

    $ counter -n deploys -add
    1
    $ counter -n deploys -add
    2
    $ counter -list
    deploys	2

## Installation

    go install github.com/andreimerlescu/counter@latest
    counter -h

Or build from source:

    git clone git@github.com:andreimerlescu/counter.git
    cd counter
    make install

Windows builds compile but are untested.

## Quick start

    $ counter -n subscribers
    Error: directory /tmp/.counters does not exist (use -force to create it)
    $ counter -n subscribers -F          # create the counter directory
    0
    $ counter -n subscribers -add
    1
    $ counter -n subscribers -add -q 5
    6
    $ counter -n subscribers -sub
    5
    $ counter -n subscribers -set 20
    20
    $ counter -n subscribers -set 0
    0
    $ counter -n subscribers -reset
    Error: refusing to reset counter subscribers without -yes
    $ counter -n subscribers -reset -yes
    0
    $ counter -n subscribers -delete -yes
    counter subscribers deleted
    $ counter -n subscribers             # a missing counter reads as 0
    0

A counter that doesn't exist yet reads as `0`, and the first `-add`, `-sub`, or
`-set` creates it.

## Flags

Every flag has a short and a long form. `-q 5`, `-q=5`, and `-quantity 5` are
all equivalent.

### Choosing a counter

| Short | Long     | Type   | Default          | Description                                                  |
|-------|----------|--------|------------------|--------------------------------------------------------------|
| `-n`  | `-name`  | string |                  | Counter name, stored as a file of that name inside `-dir`    |
| `-f`  | `-file`  | string |                  | Counter file path; relative paths are resolved inside `-dir` |
| `-d`  | `-dir`   | string | `/tmp/.counters` | Directory that holds counters                                |
| `-F`  | `-force` | bool   | `false`          | Create the directory if it does not exist                    |

Exactly one of `-name` or `-file` is required for counter operations. Using both
is an error.

### Operations

Only one operation is allowed per command. Combining them, such as `-add -sub`,
is an error and leaves the counter unchanged. With no operation, `counter`
prints the current value.

| Short | Long        | Type  | Default | Description                                       |
|-------|-------------|-------|---------|---------------------------------------------------|
| `-a`  | `-add`      | bool  | `false` | Add `-q` to the counter                           |
| `-s`  | `-sub`      | bool  | `false` | Subtract `-q` from the counter                    |
| `-q`  | `-quantity` | int64 | `1`     | Amount used by `-add` and `-sub`; may be negative |
| `-S`  | `-set`      | int64 |         | Set the counter to this value, including `0`      |
| `-R`  | `-reset`    | bool  | `false` | Reset the counter to 0 (requires `-yes`)          |
| `-D`  | `-delete`   | bool  | `false` | Delete the counter (requires `-yes`)              |
| `-y`  | `-yes`      | bool  | `false` | Confirm `-reset` and `-delete`                    |

Arithmetic saturates at the int64 limits instead of wrapping. Adding to
`9223372036854775807` leaves it at `9223372036854775807`.

### Listing and searching

| Short | Long      | Type   | Description                                                    |
|-------|-----------|--------|----------------------------------------------------------------|
| `-l`  | `-list`   | bool   | List every counter in `-dir` with its value                    |
| `-g`  | `-search` | string | List counters whose name contains this text (case-insensitive) |

### Information

| Short | Long       | Description                              |
|-------|------------|------------------------------------------|
| `-v`  | `-version` | Print the version                        |
| `-h`  | `-help`    | Print help to stderr                     |
|       | `-usage`   | Print help to stdout                     |
|       | `-env`     | Print the effective environment settings |

## Listing and searching counters

    $ counter -l
    Subscriptions	1000
    errors	-3
    visits	7
    $ counter -g sub
    Subscriptions	1000
    $ counter -g nothing-matches; echo "exit $?"
    exit 1

Output is one counter per line, `name<TAB>value`, sorted by name, which makes
it easy to process:

    counter -l | sort -t$'\t' -k2 -n        # sort by value
    counter -l | cut -f1                     # names only
    counter -g deploy >/dev/null && echo "found deploy counters"

A few details:

- `-search` exits with status 1 when nothing matches, like `grep`. `-list` on an
  empty directory exits 0.
- Only the top level of `-dir` is listed. Counters in subdirectories, or at
  absolute `-file` paths elsewhere, are not shown.
- Files in the directory that don't contain a counter value are skipped with a
  warning on stderr.
- Listing never creates the directory, even with `-force`, so a mistyped `-dir`
  is reported as an error.
- `-list` and `-search` cannot be combined with `-name`, `-file`, or an
  operation.

## Environment variables

Environment variables set defaults. Flags given on the command line always take
precedence.

| Variable                 | Type    | Effect               |
|--------------------------|---------|----------------------|
| `COUNTER_DIR`            | string  | Default for `-dir`   |
| `COUNTER_QUANTITY`       | int64   | Default for `-q`     |
| `COUNTER_USE_FORCE`      | boolean | Default for `-force` |
| `COUNTER_ALWAYS_YES`     | boolean | Default for `-yes`   |
| `COUNTER_NEVER_ADD`      | boolean | Refuse `-add`        |
| `COUNTER_NEVER_SUBTRACT` | boolean | Refuse `-sub`        |
| `COUNTER_NEVER_SET_TO`   | boolean | Refuse `-set`        |
| `COUNTER_NEVER_RESET`    | boolean | Refuse `-reset`      |
| `COUNTER_NEVER_DELETE`   | boolean | Refuse `-delete`     |

Booleans accept `1`, `t`, `true`, `0`, `f`, and `false`, in any case. An invalid
value, such as `COUNTER_QUANTITY=abc` or `COUNTER_USE_FORCE=yes`, is an error
and nothing is changed.

    $ export COUNTER_QUANTITY=3
    $ counter -n threes -add
    3
    $ counter -n threes -add -q 1        # the flag wins
    4
    $ counter -env
    COUNTER_DIR=/tmp/.counters
    COUNTER_QUANTITY=3
    COUNTER_USE_FORCE=false
    COUNTER_ALWAYS_YES=false
    COUNTER_NEVER_ADD=false
    COUNTER_NEVER_SUBTRACT=false
    COUNTER_NEVER_SET_TO=false
    COUNTER_NEVER_RESET=false
    COUNTER_NEVER_DELETE=false

### Guarded counters

The `COUNTER_NEVER_*` variables turn an operation into an error, which is useful
for counters that should only ever grow:

    export COUNTER_NEVER_SUBTRACT=1
    export COUNTER_NEVER_SET_TO=1
    export COUNTER_NEVER_RESET=1
    export COUNTER_NEVER_DELETE=1

With those set:

    $ counter -n visits -sub
    Error: -sub is disabled by COUNTER_NEVER_SUBTRACT
    $ counter -n visits -add
    1

These are guardrails against mistakes, not a security boundary. Anyone who can
run `counter` can unset the variables or edit the counter file directly.

## Exit codes

| Code | Meaning                                                                 |
|------|-------------------------------------------------------------------------|
| `0`  | Success                                                                 |
| `1`  | The operation failed or was refused, or `-search` found no matches      |
| `2`  | Invalid flags, unexpected arguments, or an invalid environment variable |

## Storage

- **Location.** A counter named `visits` is stored at `<dir>/visits`. With
  `-file`, the path is used as given; relative paths are resolved inside
  `-dir`, and symlinks are followed.
- **Names.** A name must be a single file name. It can't contain `/` or `\`,
  can't contain control characters, and can't be `.` or `..`.
- **Format.** The file holds the value in decimal followed by a newline, so
  `cat` works on it.
- **Permissions.** Counter files are created with mode `0600`. Directories
  created with `-force` get mode `0700`.
- **Concurrency.** Every read, update, and delete takes a file lock, so
  concurrent `-add` calls from many processes never lose an update.
- **Crash safety.** Values are overwritten in place without truncating first,
  so an interrupted write leaves the old value or the new one, never an empty
  file.

The default directory, `/tmp/.counters`, is cleared on reboot on many systems,
and other users on the machine can see it. For counters you care about, point
`COUNTER_DIR` at a directory you own:

    export COUNTER_DIR="$HOME/.local/state/counters"
    counter -n visits -F

## Upgrading from 1.0.x

Version 1.1.0 fixes several bugs, and some of those fixes change behavior that
scripts may depend on:

| Before (1.0.x)                                                  | Now (1.1.0)                                                                  |
|-----------------------------------------------------------------|------------------------------------------------------------------------------|
| Environment variables overrode flags                            | Flags override environment variables                                         |
| Booleans in the environment only accepted `1`                   | `1`, `t`, `true`, `0`, `f`, `false` in any case; invalid values are an error |
| `-set 0` was ignored                                            | `-set 0` sets the counter to 0                                               |
| `-add` past the int64 limit wrapped to a huge negative number   | It saturates at the limit                                                    |
| Combined operations (`-add -sub`) were silently merged          | Combined operations are an error                                             |
| `COUNTER_NEVER_*` silently skipped the operation                | It is an error with a non-zero exit code                                     |
| `-delete` exited 1 on success, and claimed success on failure   | Exits 0 on success; deleting a missing counter is an error                   |
| `-name` and `-file` could be combined                           | Using both is an error                                                       |
| Concurrent updates could lose increments                        | Updates are locked                                                           |
| Counter files were read-only (`0444`)                           | Files are `0600`; old files are fixed on their next update                   |
| Counters from 1.0.2 and earlier used hashed file names          | They are moved to plain names automatically on first use                     |
| `-version` was documented but did not exist                     | `-version` works                                                             |

### Counters created by 1.0.2 and earlier

Version 1.0.2 and earlier stored named counters under a hashed file name, such
as `.named.c17be803540fe11391c1714f.counter`. Counter moves these to the new
layout automatically: the first time you use a counter by name (`-n NAME`),
its old file is renamed to `NAME` and carries on from its old value. No action
is needed.

Until a counter has been used by name once, `-list` shows it under its hashed
file name, because the original name can't be recovered from the hash.

If a counter exists under both names, for example because you used 1.0.2 and
then 1.0.3, the plain-name file is used and the hashed file is left untouched.
Compare the two with `cat` and delete whichever you don't need.

Avoid running a 1.0.x binary and 1.1.0 against the same counters at the same
time. The old binary doesn't lock files and writes to the hashed names.

## Development

    go test ./...          # full suite, including multi-process concurrency tests
    go test -short ./...   # skips the concurrency tests

The tests run the real CLI end to end. The test binary re-executes itself as
`counter`, so flag parsing, environment handling, file locking, legacy
migration, and exit codes are all covered.