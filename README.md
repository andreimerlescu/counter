# Counter

This application is designed to be a counter on your linux/unix/macOS filesystem that can be invoked with a bunch
of custom options. 

## Installation

```bash
go install github.com/andreimerlescu/counter@latest
counter -h
```

## Usage

```log
Usage of counter:
  -F    force overwrite
  -a    add 1 to the counter (default true)
  -add
        add 1 to the counter (default true)
  -d string
        counter directory (default "/tmp/.counters")
  -delete
        remove counter (requires -yes)
  -dir string
        counter directory (default "/tmp/.counters")
  -directory string
        counter directory (default "/tmp/.counters")
  -f string
        counter file name (default "/tmp/.counters/default")
  -file string
        counter file name (default "/tmp/.counters/default")
  -force
        force overwrite
  -n string
        counter name (default "default")
  -name string
        counter name (default "default")
  -s    subtract 1 from the counter
  -sub
        subtract 1 from the counter
  -subtract
        subtract 1 from the counter
  -v    show version
  -ver
        show version
  -version
        show version
  -yes
        your response is yes
```

## Examples

Set your global choice for `-dir=` with `COUNTER_DIR` environment variable.

```bash
echo "echo 'export COUNTER_DIR=\"$HOME/.counters\"' | tee -a ~/.bashrc > /dev/null
```

```zsh
echo "echo 'export COUNTER_DIR=\"$HOME/.counters\"' | tee -a ~/.zshrc > /dev/null
```

You can also set `COUNTER_NEVER_DELETE=1` if you wish to disable os.Remove() functionality.
Finally, the last ENV you can define that is accepted is the `COUNTER_USE_FORCE=1` which 
always uses `-F` in case the directory the counter will live in does not exist.

## Real World Examples

```bash
$ go install github.com/andreimerlescu/counter@latest
$ export COUNTER_USE_FORCE=1
$ export COUNTER_DIR="${HOME}/.counters"
$ counter -name test
0
$ counter -name test -add 
1
$ counter -name test -sub
0
$ counter -name test -delete
deleting counter test at 0 - confirm by re-running with -yes
$ counter -name test -delete -yes
$ counter -name test 
0
```

> ###  🚨🚨🚨 NOTE 🚨🚨🚨
> This looks like an error, but it is not actually. When `COUNTER_USE_FORCE` is enabled and `-name=` is requesting
> a non-existent counter, it'll create the counter for you. Watch...

> #### 2ND NOTE 
> Since `COUNTER_NEVER_DELETE` is unset here, the `-delete` flag is accepted and the counter is removed. The
> `-delete` flag depends on the `-yes` flag in order to execute `os.Remove()` on the counters.

```bash
$ counter -name test -delete
$ unset COUNTER_USE_FORCE
$ counter -name test
no such counter test
$ counter -name test -add
1
$ counter -name test
1
```

> ### **NOTE** 
> When `COUNTER_USE_FORCE` is not enabled, a GET request using just `-name=` will be rejected.

```bash
$ export COUNTER_NEVER_DELETE=1
$ counter -name test
1
$ counter -name test -delete
you must first unset COUNTER_NEVER_DELETE before running -delete
$ unset COUNTER_NEVER_DELETE
$ counter -name test -delete
deleting counter test at 0 - confirm by re-running with -yes
$ counter -name test
1
$ counter -name test -delete -yes
counter test deleted
$ counter -name test
no such counter test
$ counter -name test -sub
-1
$ counter -name test -add 
0
$ for i in $(seq 1 434); do counter -name test -add; done


```

## Building

```bash
git clone git@github.com:andreimerlescu/counter.git
cd counter
make install
counter -h
```

## Testing

```bash
go test ./...
```

```log
=== RUN   TestGenerateCounterFileName
--- PASS: TestGenerateCounterFileName (0.00s)
=== RUN   TestEnsureDir
--- PASS: TestEnsureDir (0.00s)
=== RUN   TestReadCounter
--- PASS: TestReadCounter (0.00s)
=== RUN   TestWriteCounter
--- PASS: TestWriteCounter (0.00s)
=== RUN   TestSetUnsetImmutable
--- PASS: TestSetUnsetImmutable (0.00s)
PASS

Process finished with the exit code 
```

### Benchmark Performance

```log
goos: linux
goarch: amd64
pkg: github.com/andreimerlescu/countable
cpu: Intel(R) Xeon(R) W-3245 CPU @ 3.20GHz
BenchmarkWriteCounter
BenchmarkWriteCounter-9              	  488050	      2274 ns/op
BenchmarkReadCounter
BenchmarkReadCounter-9               	  234050	      4797 ns/op
BenchmarkGenerateCounterFileName
BenchmarkGenerateCounterFileName-9   	 1955614	       611.2 ns/op
BenchmarkEnsureDir
BenchmarkEnsureDir-9                 	   67807	     17288 ns/op
BenchmarkResolveSymlink
BenchmarkResolveSymlink-9            	  183456	      6180 ns/op
PASS

Process finished with the exit code 0
```