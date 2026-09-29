package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

const (
	DefaultCounterDir = "/tmp/.counters"
	DefaultQuantity   = int64(1)

	// maxRetries bounds how often an operation restarts because the counter
	// file was deleted or replaced while we were waiting for its lock.
	maxRetries = 10
)

// errEmptyCounter means the file exists but holds no value yet. That only
// happens between a writer creating the file and writing its first value.
var errEmptyCounter = errors.New("counter file is empty")

type config struct {
	dir      string
	name     string
	file     string
	quantity int64
	setTo    int64
	setGiven bool

	add, sub, reset, del bool
	force, yes           bool

	neverAdd, neverSub, neverSet, neverReset, neverDelete bool

	showVersion, showUsage, showEnv bool

	list   bool
	search string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the whole CLI. Precedence is: defaults < environment < flags.
func run(args []string, stdout, stderr io.Writer) int {
	c := &config{dir: DefaultCounterDir, quantity: DefaultQuantity}

	for _, b := range c.envBindings() {
		if v := os.Getenv(b.name); v != "" {
			if err := b.set(v); err != nil {
				fmt.Fprintf(stderr, "Error: invalid %s=%q: %v\n", b.name, v, err)
				return 2
			}
		}
	}

	flags := c.flagSet(stderr)
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2 // the flag package has already printed the error and usage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "Error: unexpected arguments: %s\n", strings.Join(flags.Args(), " "))
		return 2
	}
	// 0 is a valid value for -set, so detect whether it was given rather
	// than treating 0 as "not set".
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "S" || f.Name == "set" {
			c.setGiven = true
		}
	})

	switch {
	case c.showVersion:
		fmt.Fprintln(stdout, BinaryVersion())
		return 0
	case c.showUsage:
		flags.SetOutput(stdout)
		flags.Usage()
		return 0
	case c.showEnv:
		for _, b := range c.envBindings() {
			fmt.Fprintf(stdout, "%s=%s\n", b.name, b.get())
		}
		return 0
	}

	if c.list || c.search != "" {
		found, err := c.listCounters(stdout, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		if !found && c.search != "" {
			return 1 // like grep: no match is a non-zero exit
		}
		return 0
	}

	if err := c.execute(stdout); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

// ---------------------------------------------------------------------------
// Configuration: environment and flags
// ---------------------------------------------------------------------------

type envBinding struct {
	name string
	desc string
	set  func(string) error
	get  func() string
}

func stringEnv(name, desc string, p *string) envBinding {
	return envBinding{name, desc,
		func(v string) error { *p = v; return nil },
		func() string { return *p }}
}

func int64Env(name, desc string, p *int64) envBinding {
	return envBinding{name, desc,
		func(v string) error {
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return err
			}
			*p = n
			return nil
		},
		func() string { return strconv.FormatInt(*p, 10) }}
}

func boolEnv(name, desc string, p *bool) envBinding {
	return envBinding{name, desc,
		func(v string) error {
			b, err := strconv.ParseBool(v)
			if err != nil {
				return err
			}
			*p = b
			return nil
		},
		func() string { return strconv.FormatBool(*p) }}
}

func (c *config) envBindings() []envBinding {
	return []envBinding{
		stringEnv("COUNTER_DIR", "default for -dir", &c.dir),
		int64Env("COUNTER_QUANTITY", "default for -q", &c.quantity),
		boolEnv("COUNTER_USE_FORCE", "default for -force", &c.force),
		boolEnv("COUNTER_ALWAYS_YES", "default for -yes", &c.yes),
		boolEnv("COUNTER_NEVER_ADD", "refuse -add", &c.neverAdd),
		boolEnv("COUNTER_NEVER_SUBTRACT", "refuse -sub", &c.neverSub),
		boolEnv("COUNTER_NEVER_SET_TO", "refuse -set", &c.neverSet),
		boolEnv("COUNTER_NEVER_RESET", "refuse -reset", &c.neverReset),
		boolEnv("COUNTER_NEVER_DELETE", "refuse -delete", &c.neverDelete),
	}
}

// flagSet registers every flag with the current (env-derived) values as
// defaults, so flags given on the command line win over the environment.
func (c *config) flagSet(output io.Writer) *flag.FlagSet {
	flags := flag.NewFlagSet("counter", flag.ContinueOnError)
	flags.SetOutput(output)

	boolPair := func(short, long string, p *bool, usage string) {
		flags.BoolVar(p, short, *p, usage)
		flags.BoolVar(p, long, *p, usage)
	}
	stringPair := func(short, long string, p *string, usage string) {
		flags.StringVar(p, short, *p, usage)
		flags.StringVar(p, long, *p, usage)
	}
	int64Pair := func(short, long string, p *int64, usage string) {
		flags.Int64Var(p, short, *p, usage)
		flags.Int64Var(p, long, *p, usage)
	}

	stringPair("n", "name", &c.name, "counter name; stored as a file of that name inside -dir")
	stringPair("f", "file", &c.file, "counter file path; relative paths are resolved inside -dir")
	stringPair("d", "dir", &c.dir, "directory that holds counters")
	boolPair("a", "add", &c.add, "add -q to the counter")
	boolPair("s", "sub", &c.sub, "subtract -q from the counter")
	int64Pair("q", "quantity", &c.quantity, "amount used by -add and -sub")
	int64Pair("S", "set", &c.setTo, "set the counter to this value")
	boolPair("R", "reset", &c.reset, "reset the counter to 0 (requires -yes)")
	boolPair("D", "delete", &c.del, "delete the counter (requires -yes)")
	boolPair("F", "force", &c.force, "create -dir if it does not exist")
	boolPair("y", "yes", &c.yes, "confirm -reset and -delete")
	boolPair("v", "version", &c.showVersion, "print the version")
	flags.BoolVar(&c.showUsage, "usage", false, "print this help to stdout")
	flags.BoolVar(&c.showEnv, "env", false, "print the effective environment settings")
	boolPair("l", "list", &c.list, "list counters in -dir with their values")
	stringPair("g", "search", &c.search, "list counters in -dir whose name contains this text (case-insensitive)")

	// Usage is generated from the registered flags, so it cannot drift.
	flags.Usage = func() {
		w := flags.Output()
		fmt.Fprintf(w, "counter %s: a persistent integer counter stored in a file\n\n", BinaryVersion())
		fmt.Fprint(w, "Usage:\n"+
			"  counter (-n NAME | -f FILE) [-add | -sub | -set N | -reset | -delete] [options]\n"+
			"  counter [-l | -list] [(-g | -search) TEXT] [-dir DIR]\n\n")
		fmt.Fprint(w, "Flags:\n")
		flags.PrintDefaults()
		fmt.Fprint(w, "\nEnvironment (flags take precedence; booleans accept 1/0 or true/false):\n")
		for _, b := range c.envBindings() {
			fmt.Fprintf(w, "  %-24s %s\n", b.name, b.desc)
		}
		fmt.Fprint(w, `
Examples:
  $ counter -n visits -add
  1
  $ counter -n visits -add -q 5
  6
  $ counter -n visits -set 0
  0
  $ counter -n visits -delete -yes
  counter visits deleted
  $ counter -list
  subscriptions	1000
  visits	6
  $ counter -search SUB
  subscriptions	1000
`)
	}
	return flags
}

// ---------------------------------------------------------------------------
// Operations
// ---------------------------------------------------------------------------

func (c *config) execute(stdout io.Writer) error {
	ops := 0
	for _, on := range []bool{c.add, c.sub, c.setGiven, c.reset, c.del} {
		if on {
			ops++
		}
	}
	if ops > 1 {
		return errors.New("choose only one of -add, -sub, -set, -reset, -delete")
	}

	path, label, err := c.counterPath()
	if err != nil {
		return err
	}

	var op func(int64) int64
	switch {
	case ops == 0:
		v, err := readCounter(path)
		if errors.Is(err, errEmptyCounter) {
			v, err = 0, nil // another process just created it; value not written yet
		}
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, v)
		return nil

	case c.del:
		if c.neverDelete {
			return errors.New("-delete is disabled by COUNTER_NEVER_DELETE")
		}
		if !c.yes {
			return fmt.Errorf("refusing to delete counter %s without -yes", label)
		}
		if err := deleteCounter(path, label); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "counter %s deleted\n", label)
		return nil

	case c.add:
		if c.neverAdd {
			return errors.New("-add is disabled by COUNTER_NEVER_ADD")
		}
		op = func(v int64) int64 { return addSaturating(v, c.quantity) }

	case c.sub:
		if c.neverSub {
			return errors.New("-sub is disabled by COUNTER_NEVER_SUBTRACT")
		}
		op = func(v int64) int64 { return subSaturating(v, c.quantity) }

	case c.setGiven:
		if c.neverSet {
			return errors.New("-set is disabled by COUNTER_NEVER_SET_TO")
		}
		op = func(int64) int64 { return c.setTo }

	case c.reset:
		if c.neverReset {
			return errors.New("-reset is disabled by COUNTER_NEVER_RESET")
		}
		if !c.yes {
			return fmt.Errorf("refusing to reset counter %s without -yes", label)
		}
		op = func(int64) int64 { return 0 }
	}

	v, err := updateCounter(path, op)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, v)
	return nil
}

// counterPath works out which file the counter lives in and makes sure its
// directory exists.
func (c *config) counterPath() (path, label string, err error) {
	switch {
	case c.name != "" && c.file != "":
		return "", "", errors.New("-name and -file cannot be used together")
	case c.name != "":
		if err := validateName(c.name); err != nil {
			return "", "", err
		}
		path, label = filepath.Join(c.dir, c.name), c.name
	case c.file != "":
		path, label = c.file, c.file
		if !filepath.IsAbs(path) {
			path = filepath.Join(c.dir, path)
		}
	default:
		return "", "", errors.New("-name or -file is required")
	}

	if err := ensureDir(filepath.Dir(path), c.force); err != nil {
		return "", "", err
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	return path, label, nil
}

// listCounters prints "name<TAB>value" for every counter in c.dir, sorted by
// name, optionally filtered by a case-insensitive substring. It reports
// whether anything was printed.
func (c *config) listCounters(stdout, stderr io.Writer) (bool, error) {
	if c.name != "" || c.file != "" {
		return false, errors.New("-list and -search cannot be combined with -name or -file")
	}
	if c.add || c.sub || c.setGiven || c.reset || c.del {
		return false, errors.New("-list and -search cannot be combined with -add, -sub, -set, -reset, -delete")
	}
	// Never create the directory just to list it; a typo in -dir should be an error.
	if err := ensureDir(c.dir, false); err != nil {
		return false, err
	}

	entries, err := os.ReadDir(c.dir) // sorted by file name
	if err != nil {
		return false, fmt.Errorf("failed to list %s: %w", c.dir, err)
	}

	needle := strings.ToLower(c.search)
	found := false
	for _, e := range entries {
		name := e.Name()
		if needle != "" && !strings.Contains(strings.ToLower(name), needle) {
			continue
		}
		path := filepath.Join(c.dir, name)
		info, err := os.Stat(path) // follows symlinks, like -name does
		if err != nil || !info.Mode().IsRegular() {
			continue // subdirectories, broken symlinks, sockets, or already gone
		}

		data, exists, err := readLocked(path)
		if err != nil {
			fmt.Fprintf(stderr, "Warning: skipping %s: %v\n", displayName(name), err)
			continue
		}
		if !exists {
			continue // deleted after ReadDir
		}
		v, err := parseCounter(data)
		if errors.Is(err, errEmptyCounter) {
			continue // being created; it has no committed value yet
		}
		if err != nil {
			fmt.Fprintf(stderr, "Warning: skipping %s: %v\n", displayName(name), err)
			continue
		}
		fmt.Fprintf(stdout, "%s\t%d\n", displayName(name), v)
		found = true
	}
	return found, nil
}

// validateName keeps -name a single, printable file name inside the counter
// directory. Control characters would break -list's one-per-line output.
func validateName(name string) error {
	if name == "" || name == "." || name == ".." ||
		strings.ContainsAny(name, `/\`) ||
		strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return fmt.Errorf("invalid counter name %q: must be a single file name without path separators or control characters", name)
	}
	return nil
}

// displayName quotes names that would corrupt line-oriented output. New
// counters can't have such names, but files placed in the directory by
// other means might.
func displayName(name string) string {
	if strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return strconv.Quote(name)
	}
	return name
}

func addSaturating(a, b int64) int64 {
	switch {
	case b > 0 && a > math.MaxInt64-b:
		return math.MaxInt64
	case b < 0 && a < math.MinInt64-b:
		return math.MinInt64
	}
	return a + b
}

func subSaturating(a, b int64) int64 {
	if b == math.MinInt64 { // -b is not representable
		if a >= 0 {
			return math.MaxInt64
		}
		return a - b // exact: result is in [0, MaxInt64]
	}
	return addSaturating(a, -b)
}

// ---------------------------------------------------------------------------
// File storage
//
// Every access takes a lock on the counter file itself (shared for reads,
// exclusive for updates and deletes) and then checks that the locked file is
// still the one at path. That check catches files deleted or replaced while
// we waited, so nobody reads or writes an unlinked file.
// ---------------------------------------------------------------------------

// ensureDir ensures that a directory exists, creating it (private to the
// user) only when force is set.
func ensureDir(dir string, force bool) error {
	info, err := os.Stat(dir)
	switch {
	case err == nil:
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", dir)
		}
		return nil
	case !errors.Is(err, fs.ErrNotExist):
		return err
	case !force:
		return fmt.Errorf("directory %s does not exist (use -force to create it)", dir)
	}
	return os.MkdirAll(dir, 0o700)
}

// readCounter reads the counter under a shared lock. A missing file is 0.
func readCounter(path string) (int64, error) {
	data, exists, err := readLocked(path)
	if err != nil || !exists {
		return 0, err
	}
	return parseCounter(data)
}

// readLocked returns the contents of the file at path, read under a shared
// lock. exists is false if there is no file, including when it is deleted
// while we wait for the lock. If it is replaced while we wait, the
// replacement is read instead.
func readLocked(path string) (data []byte, exists bool, err error) {
	for i := 0; i < maxRetries; i++ {
		f, openErr := os.Open(path)
		if errors.Is(openErr, fs.ErrNotExist) {
			return nil, false, nil
		}
		if openErr != nil {
			return nil, false, fmt.Errorf("failed to open counter file: %w", openErr)
		}
		data, stale, readErr := readOpen(f, path)
		_ = f.Close()
		if stale {
			continue
		}
		if readErr != nil {
			return nil, false, readErr
		}
		return data, true, nil
	}
	return nil, false, errors.New("counter file was repeatedly replaced during read; giving up")
}

func readOpen(f *os.File, path string) (data []byte, stale bool, err error) {
	if err = lockFile(f, false); err != nil {
		return nil, false, fmt.Errorf("failed to lock counter file: %w", err)
	}
	if stale, err = isStale(f, path); stale || err != nil {
		return nil, stale, err
	}
	if data, err = io.ReadAll(f); err != nil {
		return nil, false, fmt.Errorf("failed to read counter file: %w", err)
	}
	return data, false, nil
}

func parseCounter(data []byte) (int64, error) {
	s := strings.TrimSpace(string(data))
	if s == "" {
		return 0, errEmptyCounter
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid counter value %q: %w", s, err)
	}
	return v, nil
}

// updateCounter applies op to the stored value while holding an exclusive
// lock across the whole read-modify-write.
func updateCounter(path string, op func(int64) int64) (int64, error) {
	for i := 0; i < maxRetries; i++ {
		f, err := openForUpdate(path)
		if err != nil {
			return 0, fmt.Errorf("failed to open counter file: %w", err)
		}
		v, stale, err := updateLocked(f, path, op)
		closeErr := f.Close()
		switch {
		case stale:
			continue
		case err != nil:
			return 0, err
		case closeErr != nil:
			return 0, fmt.Errorf("failed to close counter file: %w", closeErr)
		}
		return v, nil
	}
	return 0, errors.New("counter file was repeatedly replaced during update; giving up")
}

func openForUpdate(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if errors.Is(err, fs.ErrPermission) {
		// Versions up to 1.0.3 left counter files read-only (0444).
		if _, statErr := os.Stat(path); statErr == nil && os.Chmod(path, 0o600) == nil {
			f, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
		}
	}
	return f, err
}

func updateLocked(f *os.File, path string, op func(int64) int64) (v int64, stale bool, err error) {
	if err = lockFile(f, true); err != nil {
		return 0, false, fmt.Errorf("failed to lock counter file: %w", err)
	}
	if stale, err = isStale(f, path); stale || err != nil {
		return 0, stale, err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return 0, false, fmt.Errorf("failed to read counter file: %w", err)
	}
	old, err := parseCounter(data)
	if errors.Is(err, errEmptyCounter) {
		old, err = 0, nil // we just created it
	}
	if err != nil {
		return 0, false, err
	}
	v = op(old)
	if err := writeInPlace(f, len(data), v); err != nil {
		return 0, false, fmt.Errorf("failed to write counter file: %w", err)
	}
	return v, false, nil
}

// isStale reports whether the file we locked is no longer the one at path,
// because it was deleted (and possibly recreated) while we waited.
func isStale(f *os.File, path string) (bool, error) {
	held, err := f.Stat()
	if err != nil {
		return false, err
	}
	current, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return !os.SameFile(held, current), nil
}

// writeInPlace overwrites the value without truncating first. The new value
// is padded with spaces to cover the old one, so a crash at any point leaves
// a parseable file (old value, or new value plus padding) rather than an
// empty one.
func writeInPlace(f *os.File, oldLen int, v int64) error {
	s := strconv.FormatInt(v, 10) + "\n"
	buf := s
	if pad := oldLen - len(s); pad > 0 {
		buf += strings.Repeat(" ", pad)
	}
	if _, err := f.WriteAt([]byte(buf), 0); err != nil {
		return err
	}
	if err := f.Truncate(int64(len(s))); err != nil {
		return err
	}
	return f.Sync()
}

func deleteCounter(path, label string) error {
	for i := 0; i < maxRetries; i++ {
		f, err := os.Open(path)
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("counter %s does not exist", label)
		}
		if err != nil {
			return err
		}
		err = lockFile(f, true)
		var stale bool
		if err == nil {
			stale, err = isStale(f, path)
		}
		if err == nil && !stale {
			err = os.Remove(path)
		}
		_ = f.Close()
		if !stale {
			return err
		}
	}
	return errors.New("counter file was repeatedly replaced during delete; giving up")
}
