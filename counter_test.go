package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Harness: re-exec the test binary as the counter CLI
// ---------------------------------------------------------------------------

const runMainEnv = "GO_TEST_RUN_COUNTER_MAIN"

// TestMain lets the test binary act as the counter CLI when re-executed by the
// tests below. This exercises the real main() (flag parsing, env handling,
// file I/O, exit codes) end to end.
func TestMain(m *testing.M) {
	if os.Getenv(runMainEnv) == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type result struct {
	stdout string
	stderr string
	code   int
}

// childEnv returns a clean environment: no inherited COUNTER_* variables, so
// the developer's shell can't influence test outcomes.
func childEnv(extra map[string]string) []string {
	env := []string{runMainEnv + "=1"}
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "COUNTER_") || strings.HasPrefix(kv, runMainEnv+"=") {
			continue
		}
		env = append(env, kv)
	}
	for k, v := range extra {
		env = append(env, k+"="+v)
	}
	return env
}

// execCounter runs the CLI and never calls t.Fatal, so it is safe from goroutines.
func execCounter(env map[string]string, args ...string) (result, error) {
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = childEnv(env)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	r := result{stdout: stdout.String(), stderr: stderr.String()}
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return r, err
		}
		r.code = exitErr.ExitCode()
	}
	return r, nil
}

func runCounter(t testing.TB, env map[string]string, args ...string) result {
	t.Helper()
	r, err := execCounter(env, args...)
	if err != nil {
		t.Fatalf("could not run counter %v: %v", args, err)
	}
	return r
}

func mustRun(t testing.TB, env map[string]string, args ...string) result {
	t.Helper()
	r := runCounter(t, env, args...)
	if r.code != 0 {
		t.Fatalf("counter %v: exit %d, stderr: %s", args, r.code, strings.TrimSpace(r.stderr))
	}
	return r
}

func (r result) value(t testing.TB) int64 {
	t.Helper()
	v, err := strconv.ParseInt(strings.TrimSpace(r.stdout), 10, 64)
	if err != nil {
		t.Fatalf("stdout %q is not a counter value (exit %d, stderr %q)", r.stdout, r.code, r.stderr)
	}
	return v
}

// named builds args for a counter in dir. Every test passes -d so nothing
// ever touches the real default directory.
func named(dir, name string, args ...string) []string {
	return append([]string{"-d", dir, "-n", name}, args...)
}

func seed(t testing.TB, dir, name string, v int64) {
	t.Helper()
	if v == 0 {
		t.Fatal("seed: use a non-zero value; -S=0 is itself under test")
	}
	if got := mustRun(t, nil, named(dir, name, "-S="+strconv.FormatInt(v, 10))...).value(t); got != v {
		t.Fatalf("seed: wanted %d, got %d", v, got)
	}
}

func readValue(t testing.TB, dir, name string) int64 {
	t.Helper()
	return mustRun(t, nil, named(dir, name)...).value(t)
}

// soleFile returns the single file in dir, without depending on how the
// filename is derived from the counter name.
func soleFile(t testing.TB, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 file in %s, found %d", dir, len(entries))
	}
	return filepath.Join(dir, entries[0].Name())
}

// ---------------------------------------------------------------------------
// Basic behavior
// ---------------------------------------------------------------------------

func TestMissingCounterReadsAsZero(t *testing.T) {
	if got := readValue(t, t.TempDir(), "fresh"); got != 0 {
		t.Errorf("want 0, got %d", got)
	}
}

func TestAddAndSubtract(t *testing.T) {
	dir := t.TempDir()
	steps := []struct {
		args []string
		want int64
	}{
		{[]string{"-add"}, 1},
		{[]string{"-add", "-q=5"}, 6},
		{[]string{"-sub"}, 5},
		{[]string{"-sub", "-q=10"}, -5},
	}
	for _, s := range steps {
		if got := mustRun(t, nil, named(dir, "c", s.args...)...).value(t); got != s.want {
			t.Fatalf("%v: want %d, got %d", s.args, s.want, got)
		}
	}
	if got := readValue(t, dir, "c"); got != -5 {
		t.Errorf("persisted value: want -5, got %d", got)
	}
}

func TestResetRequiresYes(t *testing.T) {
	dir := t.TempDir()
	seed(t, dir, "c", 5)

	if r := runCounter(t, nil, named(dir, "c", "-reset")...); r.code == 0 {
		t.Errorf("-reset without -yes should fail")
	}
	if got := readValue(t, dir, "c"); got != 5 {
		t.Errorf("counter changed without -yes: got %d", got)
	}
	if got := mustRun(t, nil, named(dir, "c", "-reset", "-yes")...).value(t); got != 0 {
		t.Errorf("-reset -yes: want 0, got %d", got)
	}
}

func TestVersionFlags(t *testing.T) {
	for _, f := range []string{"-v", "-version"} {
		t.Run(f, func(t *testing.T) {
			r := runCounter(t, nil, f)
			if r.code != 0 {
				t.Fatalf("exit %d, stderr: %s", r.code, strings.TrimSpace(r.stderr))
			}
			if got := strings.TrimSpace(r.stdout); got != BinaryVersion() {
				t.Errorf("want %q, got %q", BinaryVersion(), got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Arithmetic
// ---------------------------------------------------------------------------

func TestArithmeticSaturatesAtBounds(t *testing.T) {
	cases := []struct {
		name  string
		start int64
		args  []string
		want  int64
	}{
		{"add at max", math.MaxInt64, []string{"-add"}, math.MaxInt64},
		{"add negative quantity at min", math.MinInt64, []string{"-add", "-q=-1"}, math.MinInt64},
		{"sub at min", math.MinInt64, []string{"-sub"}, math.MinInt64},
		{"sub negative quantity at max", math.MaxInt64, []string{"-sub", "-q=-1"}, math.MaxInt64},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			seed(t, dir, "c", tc.start)
			if got := mustRun(t, nil, named(dir, "c", tc.args...)...).value(t); got != tc.want {
				t.Errorf("want %d, got %d (wrapped around)", tc.want, got)
			}
		})
	}
}

func TestSetToZero(t *testing.T) {
	for _, f := range []string{"-S=0", "-set=0"} {
		t.Run(f, func(t *testing.T) {
			dir := t.TempDir()
			seed(t, dir, "c", 5)
			if got := mustRun(t, nil, named(dir, "c", f)...).value(t); got != 0 {
				t.Errorf("want 0, got %d", got)
			}
			if got := readValue(t, dir, "c"); got != 0 {
				t.Errorf("persisted value: want 0, got %d", got)
			}
		})
	}
}

func TestConflictingOperationsRejected(t *testing.T) {
	cases := [][]string{
		{"-add", "-sub"},
		{"-add", "-S=10"},
		{"-reset", "-add", "-yes"},
		{"-delete", "-add", "-yes"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			dir := t.TempDir()
			seed(t, dir, "c", 5)
			if r := runCounter(t, nil, named(dir, "c", args...)...); r.code == 0 {
				t.Errorf("expected non-zero exit for conflicting flags, got stdout %q", r.stdout)
			}
			if got := readValue(t, dir, "c"); got != 5 {
				t.Errorf("counter modified by rejected command: want 5, got %d", got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestDelete(t *testing.T) {
	t.Run("success exits zero", func(t *testing.T) {
		dir := t.TempDir()
		seed(t, dir, "c", 5)
		r := runCounter(t, nil, named(dir, "c", "-delete", "-yes")...)
		if r.code != 0 {
			t.Errorf("successful delete: want exit 0, got %d", r.code)
		}
		if entries, _ := os.ReadDir(dir); len(entries) != 0 {
			t.Errorf("counter file still present after delete")
		}
	})

	t.Run("failure is not reported as success", func(t *testing.T) {
		dir := t.TempDir()
		r := runCounter(t, nil, named(dir, "missing", "-delete", "-yes")...)
		if r.code == 0 {
			t.Errorf("deleting a nonexistent counter should fail")
		}
		if strings.Contains(r.stdout, "deleted") {
			t.Errorf("reported deletion even though removal failed: %q", r.stdout)
		}
	})

	t.Run("requires yes", func(t *testing.T) {
		dir := t.TempDir()
		seed(t, dir, "c", 5)
		if r := runCounter(t, nil, named(dir, "c", "-delete")...); r.code == 0 {
			t.Errorf("-delete without -yes should fail")
		}
		if got := readValue(t, dir, "c"); got != 5 {
			t.Errorf("counter changed: got %d", got)
		}
	})

	t.Run("never delete", func(t *testing.T) {
		dir := t.TempDir()
		seed(t, dir, "c", 5)
		env := map[string]string{"COUNTER_NEVER_DELETE": "1"}
		if r := runCounter(t, env, named(dir, "c", "-delete", "-yes")...); r.code == 0 {
			t.Errorf("delete should be refused when COUNTER_NEVER_DELETE=1")
		}
		if got := readValue(t, dir, "c"); got != 5 {
			t.Errorf("counter changed: got %d", got)
		}
	})
}

// ---------------------------------------------------------------------------
// Environment
// ---------------------------------------------------------------------------

func TestFlagsOverrideEnvironment(t *testing.T) {
	t.Run("quantity", func(t *testing.T) {
		env := map[string]string{"COUNTER_QUANTITY": "5"}
		if got := mustRun(t, env, named(t.TempDir(), "c", "-add", "-q=2")...).value(t); got != 2 {
			t.Errorf("-q=2 should beat COUNTER_QUANTITY=5: got %d", got)
		}
	})

	t.Run("dir", func(t *testing.T) {
		envDir, flagDir := t.TempDir(), t.TempDir()
		env := map[string]string{"COUNTER_DIR": envDir}
		mustRun(t, env, named(flagDir, "c", "-add")...)
		if entries, _ := os.ReadDir(envDir); len(entries) != 0 {
			t.Errorf("counter written to COUNTER_DIR instead of -d")
		}
		if entries, _ := os.ReadDir(flagDir); len(entries) != 1 {
			t.Errorf("counter not written to -d directory")
		}
	})

	t.Run("env applies when flag absent", func(t *testing.T) {
		env := map[string]string{"COUNTER_QUANTITY": "5"}
		if got := mustRun(t, env, named(t.TempDir(), "c", "-add")...).value(t); got != 5 {
			t.Errorf("want 5, got %d", got)
		}
	})
}

func TestEnvironmentBooleans(t *testing.T) {
	for _, v := range []string{"1", "true", "TRUE", "t"} {
		t.Run(v, func(t *testing.T) {
			dir := t.TempDir()
			seed(t, dir, "c", 5)
			env := map[string]string{"COUNTER_ALWAYS_YES": v}
			if got := mustRun(t, env, named(dir, "c", "-reset")...).value(t); got != 0 {
				t.Errorf("want 0, got %d", got)
			}
		})
	}
}

func TestInvalidEnvironmentRejected(t *testing.T) {
	cases := map[string]string{
		"COUNTER_QUANTITY":  "abc",
		"COUNTER_USE_FORCE": "banana",
	}
	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			r := runCounter(t, map[string]string{k: v}, named(t.TempDir(), "c")...)
			if r.code == 0 {
				t.Errorf("%s=%s should be rejected, got stdout %q", k, v, r.stdout)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Names and paths
// ---------------------------------------------------------------------------

func TestValidateName(t *testing.T) {
	for _, name := range []string{"visits", "with space", "日本語", ".hidden"} {
		if err := validateName(name); err != nil {
			t.Errorf("%q should be valid: %v", name, err)
		}
	}
	for _, name := range []string{"", ".", "..", "../x", "a/b", `a\b`, "a\x00b", "a\nb"} {
		if validateName(name) == nil {
			t.Errorf("%q should be rejected", name)
		}
	}
}

func TestNameCannotEscapeCounterDir(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(base, "counters")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"../escaped", "a/../../escaped2"} {
		runCounter(t, nil, "-d", dir, "-n", name, "-f", "x", "-add")
	}

	_ = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if rel, _ := filepath.Rel(dir, p); strings.HasPrefix(rel, "..") {
			t.Errorf("file created outside counter dir: %s", p)
		}
		return nil
	})
}

func TestNameAloneCannotEscapeCounterDir(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "counters")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if r := runCounter(t, nil, "-d", dir, "-n", "../escaped", "-add"); r.code == 0 {
		t.Errorf("-n ../escaped should be rejected")
	}
	if _, err := os.Stat(filepath.Join(base, "escaped")); err == nil {
		t.Errorf("file created outside counter dir")
	}
}

func TestNameAndFileAreConsistent(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, nil, named(dir, "foo", "-add")...)

	r := runCounter(t, nil, named(dir, "foo", "-f", "whatever")...)
	if r.code == 0 && strings.TrimSpace(r.stdout) != "1" {
		t.Errorf("-n foo -f whatever silently read a different counter (got %q); "+
			"it should either be rejected or refer to counter foo", strings.TrimSpace(r.stdout))
	}
}

// ---------------------------------------------------------------------------
// Permissions and directories
// ---------------------------------------------------------------------------

func TestCounterFileIsPrivate(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, nil, named(dir, "c", "-add")...)
	mustRun(t, nil, named(dir, "c", "-add")...)

	info, err := os.Stat(soleFile(t, dir))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("counter file mode: want 0600, got %#o", perm)
	}
}

func TestForceCreatesPrivateDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new")
	mustRun(t, nil, "-d", dir, "-F", "-n", "c", "-add")

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("created dir mode: want 0700, got %#o", perm)
	}
}

func TestMissingDirWithoutForceFails(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "absent")
	if r := runCounter(t, nil, named(dir, "c", "-add")...); r.code == 0 {
		t.Errorf("should fail when dir is missing and -F not given")
	}
}

// Guard for the write path: when the counter can't be written, the command
// must fail.
func TestUnwritableDirFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks don't apply to root")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	if r := runCounter(t, nil, named(dir, "c", "-add")...); r.code == 0 {
		t.Errorf("write into read-only dir should fail, got stdout %q", r.stdout)
	}
}

// ---------------------------------------------------------------------------
// Concurrency
// ---------------------------------------------------------------------------

func TestConcurrentIncrementsAreNotLost(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns many processes")
	}
	dir := t.TempDir()
	const n = 50

	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := execCounter(nil, named(dir, "hits", "-add")...)
			if err != nil {
				errs <- err
				return
			}
			if r.code != 0 {
				errs <- fmt.Errorf("exit %d: %s", r.code, strings.TrimSpace(r.stderr))
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent -add failed: %v", err)
	}

	if got := readValue(t, dir, "hits"); got != n {
		t.Errorf("want %d after %d concurrent increments, got %d (lost updates)", n, n, got)
	}
}

// A reader that opened the file just before it was deleted or replaced must
// not report the old contents once it gets the lock. The test holds the
// exclusive lock in-process, lets a reader block on it, swaps the file out,
// then releases the lock.
//
// If the reader goroutine hasn't opened the file within the sleep, it simply
// sees the post-swap state, so a slow machine can only make this pass
// vacuously, never fail spuriously.
func TestReadDuringDeleteOrReplace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows cannot remove a file while it is open")
	}

	type readResult struct {
		data   []byte
		exists bool
		err    error
	}

	run := func(t *testing.T, swap func(path string)) readResult {
		path := filepath.Join(t.TempDir(), "c")
		if err := os.WriteFile(path, []byte("5\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		holder, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := lockFile(holder, true); err != nil {
			t.Fatal(err)
		}

		done := make(chan readResult, 1)
		go func() {
			data, exists, err := readLocked(path)
			done <- readResult{data, exists, err}
		}()
		time.Sleep(100 * time.Millisecond) // let the reader open the file and block

		swap(path)
		_ = holder.Close() // releases the lock

		select {
		case r := <-done:
			return r
		case <-time.After(5 * time.Second):
			t.Fatal("reader never returned")
			return readResult{}
		}
	}

	t.Run("deleted", func(t *testing.T) {
		r := run(t, func(path string) {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		})
		if r.err != nil {
			t.Fatalf("unexpected error: %v", r.err)
		}
		if r.exists {
			t.Errorf("read a deleted counter: got %q", r.data)
		}
	})

	t.Run("replaced", func(t *testing.T) {
		r := run(t, func(path string) {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("9\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		})
		if r.err != nil {
			t.Fatalf("unexpected error: %v", r.err)
		}
		if !r.exists || strings.TrimSpace(string(r.data)) != "9" {
			t.Errorf("want the replacement file's value 9, got exists=%v data=%q", r.exists, r.data)
		}
	})
}

// ---------------------------------------------------------------------------
// Usage
// ---------------------------------------------------------------------------

func TestUsageMatchesRegisteredFlags(t *testing.T) {
	// The flag package prints every registered flag on -h.
	help := runCounter(t, nil, "-h").stderr
	registered := map[string]bool{"h": true, "help": true} // handled by the flag package
	for _, m := range regexp.MustCompile(`(?m)^  -([\w.]+)`).FindAllStringSubmatch(help, -1) {
		if !strings.HasPrefix(m[1], "test.") { // flags belonging to the test binary
			registered[m[1]] = true
		}
	}
	if len(registered) <= 2 {
		t.Fatalf("could not parse flags from -h output:\n%s", help)
	}

	usage := mustRun(t, nil, "-usage").stdout

	t.Run("every flag is documented", func(t *testing.T) {
		for name := range registered {
			if name == "h" || name == "help" {
				continue
			}
			re := regexp.MustCompile(`(?m)(?:^|[\s|])-` + regexp.QuoteMeta(name) + `(?:[\s=|<]|$)`)
			if !re.MatchString(usage) {
				t.Errorf("flag -%s is not documented in -usage", name)
			}
		}
	})

	t.Run("every documented flag exists", func(t *testing.T) {
		for _, m := range regexp.MustCompile(`(?m)(?:^|[\s|])-([a-zA-Z]+)`).FindAllStringSubmatch(usage, -1) {
			if !registered[m[1]] {
				t.Errorf("-usage documents -%s, which is not a registered flag", m[1])
			}
		}
	})
}

// ---------------------------------------------------------------------------
// -list and -search
// ---------------------------------------------------------------------------

func TestList(t *testing.T) {
	dir := t.TempDir()
	seed(t, dir, "visits", 7)
	seed(t, dir, "Subscriptions", 1000)
	seed(t, dir, "errors", -3)
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	r := mustRun(t, nil, "-d", dir, "-list")
	want := "Subscriptions\t1000\nerrors\t-3\nvisits\t7\n"
	if r.stdout != want {
		t.Errorf("-list output:\nwant %q\ngot  %q", want, r.stdout)
	}
	if !strings.Contains(r.stderr, "notes.txt") {
		t.Errorf("expected a warning about the non-counter file, got stderr %q", r.stderr)
	}
}

func TestListSkipsCounterBeingCreated(t *testing.T) {
	dir := t.TempDir()
	seed(t, dir, "visits", 7)
	// An empty file is what -add leaves for an instant between creating a
	// new counter and writing its first value.
	if err := os.WriteFile(filepath.Join(dir, "pending"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	r := mustRun(t, nil, "-d", dir, "-list")
	if r.stdout != "visits\t7\n" {
		t.Errorf("want only committed counters, got %q", r.stdout)
	}
	if strings.Contains(r.stderr, "pending") {
		t.Errorf("a counter being created is not an error, got stderr %q", r.stderr)
	}
}

func TestListEmptyDir(t *testing.T) {
	r := mustRun(t, nil, "-d", t.TempDir(), "-list")
	if r.stdout != "" {
		t.Errorf("want no output for an empty dir, got %q", r.stdout)
	}
}

func TestListUsesCounterDirFromEnvironment(t *testing.T) {
	dir := t.TempDir()
	seed(t, dir, "visits", 7)
	r := mustRun(t, map[string]string{"COUNTER_DIR": dir}, "-list")
	if r.stdout != "visits\t7\n" {
		t.Errorf("got %q", r.stdout)
	}
}

func TestListMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "absent")
	if r := runCounter(t, nil, "-d", dir, "-F", "-list"); r.code == 0 {
		t.Errorf("-list on a missing dir should fail")
	}
	if _, err := os.Stat(dir); err == nil {
		t.Errorf("-list must not create the directory, even with -F")
	}
}

func TestSearch(t *testing.T) {
	dir := t.TempDir()
	seed(t, dir, "visits", 7)
	seed(t, dir, "Subscriptions", 1000)
	seed(t, dir, "subtotal", 5)

	cases := map[string]string{
		"sub": "Subscriptions\t1000\nsubtotal\t5\n",
		"SUB": "Subscriptions\t1000\nsubtotal\t5\n",
		"its": "visits\t7\n",
	}
	for text, want := range cases {
		t.Run(text, func(t *testing.T) {
			r := mustRun(t, nil, "-d", dir, "-search", text)
			if r.stdout != want {
				t.Errorf("want %q, got %q", want, r.stdout)
			}
		})
	}

	t.Run("no match exits 1", func(t *testing.T) {
		r := runCounter(t, nil, "-d", dir, "-search", "nope")
		if r.code != 1 {
			t.Errorf("want exit 1, got %d", r.code)
		}
		if r.stdout != "" {
			t.Errorf("want no output, got %q", r.stdout)
		}
	})

	t.Run("combined with -list", func(t *testing.T) {
		r := mustRun(t, nil, "-d", dir, "-list", "-search", "its")
		if r.stdout != "visits\t7\n" {
			t.Errorf("got %q", r.stdout)
		}
	})
}

func TestListConflictsRejected(t *testing.T) {
	dir := t.TempDir()
	seed(t, dir, "x", 5)
	cases := [][]string{
		{"-list", "-n", "x"},
		{"-search", "x", "-f", "x"},
		{"-list", "-add"},
		{"-search", "x", "-delete", "-yes"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if r := runCounter(t, nil, append([]string{"-d", dir}, args...)...); r.code == 0 {
				t.Errorf("expected rejection, got stdout %q", r.stdout)
			}
			if got := readValue(t, dir, "x"); got != 5 {
				t.Errorf("counter modified: want 5, got %d", got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Unit tests for storage helpers
// ---------------------------------------------------------------------------

func TestReadCounter(t *testing.T) {
	t.Run("missing file is zero", func(t *testing.T) {
		got, err := readCounter(filepath.Join(t.TempDir(), "nope"))
		if err != nil || got != 0 {
			t.Errorf("want (0, nil), got (%d, %v)", got, err)
		}
	})

	cases := []struct {
		contents string
		want     int64
		wantErr  bool
	}{
		{"42", 42, false},
		{" 42\n", 42, false},
		{"-7", -7, false},
		{"9223372036854775807", math.MaxInt64, false},
		{"", 0, true},
		{"abc", 0, true},
		{"9223372036854775808", 0, true},
	}
	for _, tc := range cases {
		t.Run(strconv.Quote(tc.contents), func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "c")
			if err := os.WriteFile(p, []byte(tc.contents), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := readCounter(p)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("want %d, got %d", tc.want, got)
			}
		})
	}
}

func TestEnsureDir(t *testing.T) {
	testDir := filepath.Join(t.TempDir(), "testDir")
	if err := ensureDir(testDir, false); err == nil {
		t.Errorf("expected error for missing dir without force")
	}
	if err := ensureDir(testDir, true); err != nil {
		t.Fatalf("ensureDir with force: %v", err)
	}
	if info, err := os.Stat(testDir); err != nil || !info.IsDir() {
		t.Errorf("directory was not created")
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkReadCounter(b *testing.B) {
	p := filepath.Join(b.TempDir(), "c")
	if err := os.WriteFile(p, []byte("12345"), 0o600); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := readCounter(p); err != nil {
			b.Fatal(err)
		}
	}
}

func TestListAndSearchShorthands(t *testing.T) {
	dir := t.TempDir()
	seed(t, dir, "visits", 7)
	seed(t, dir, "subtotal", 5)

	if got, want := mustRun(t, nil, "-d", dir, "-l").stdout, mustRun(t, nil, "-d", dir, "-list").stdout; got != want {
		t.Errorf("-l: want %q, got %q", want, got)
	}
	if got, want := mustRun(t, nil, "-d", dir, "-g", "sub").stdout, mustRun(t, nil, "-d", dir, "-search", "sub").stdout; got != want {
		t.Errorf("-g: want %q, got %q", want, got)
	}
	// -s still means subtract, not search.
	if got := mustRun(t, nil, named(dir, "visits", "-s")...).value(t); got != 6 {
		t.Errorf("-s should subtract: want 6, got %d", got)
	}
}
