# Counter

This application is designed to be a counter on your linux/unix/macOS filesystem that can be invoked with a bunch
of custom options. 

## Installation

```bash
go install github.com/andreimerlescu/counter@latest
counter -h
```

## Arguments

| Argument      | Flag                | Type     | Default                   | Usage                                                            |
|---------------|---------------------|----------|---------------------------|------------------------------------------------------------------|
| `doAdd`       | `-a` or `-add`      | `bool`   | `false`                   | add `-q=N` (1) to the counter                                    |
| `doSub`       | `-s` or `-sub`      | `bool`   | `false`                   | subtract `-q=N` (1) to the counter                               |
| `setTo`       | `-S` or `-set`      | `int64`  | `0`                       | override counter value if value is not 0 - use reset to set to 0 |
| `doReset`     | `-R` or `-reset`    | `bool`   | `false`                   | set counter to 0                                                 |
| `doDelete`    | `-D` or `-delete`   | `bool`   | `false`                   | delete the counter                                               |
| `useForce`    | `-F` or `-force`    | `bool`   | `false`                   | enable directories to be created if needed                       |
| `quantity`    | `-q` or `-quantity` | `int64`  | `1`                       | value to adjust the counter on each execution                    |
| `showVersion` | `-v` or `-version`  | `bool`   | `false`                   | show the version of the utility                                  |
| `counterDir`  | `-d` or `-dir`      | `string` | `/tmp/.counters`          | directory to save counters                                       |
| `counterFile` | `-f` or `-file`     | `string` | `/tmp/.counters/default`  | path to counter file                                             |
| `counterName` | `-n` or `-name`     | `string` | `default`                 | name of the counter                                              |


## Environment Variables

| Variable                 | Default Value | Expected Value                                      | Anticipated Action                                                | 
|--------------------------|---------------|-----------------------------------------------------|-------------------------------------------------------------------|
| `COUNTER_USE_FORCE`      | `<unset>`     | `1`                                                 | Creates required directories that do not exist.                   | 
| `COUNTER_DIR`            | `<unset>`     | `[A-Za-z0-9._+/]+{3,69}`                            | Path to directory where counters are saved.                       |
| `COUNTER_NEVER_DELETE`   | `<unset>`     | `1`                                                 | Prevent os.Remove() from deleting files or directories.           |
| `COUNTER_NEVER_SET_TO`   | `<unset>`     | `1`                                                 | Prevent -S or -set usage on the counters.                         |
| `COUNTER_NEVER_SUBTRACT` | `<unset>`     | `1`                                                 | Enable positive growth only counters.                             | 
| `COUNTER_NEVER_ADD`      | `<unset>`     | `1`                                                 | Enable negative growth only counters.                             |
| `COUNTER_NEVER_RESET`    | `<unset>`     | `1`                                                 | Prevent a counter from getting reset.                             |
| `COUNTER_QUANTITY`       | `<unset>`     | `[0-9]` (valid from math.MinInt64 to math.MaxInt64) | Adjust the quantity to increase/decrease upon -add/-sub requests. | 

## Common Argument Combinations

### Create a locked down environment

1. Edit your `~/.bashrc` or `~/.zshrc` file to add: 

    ```bash
    export COUNTER_NEVER_RESET=1
    export COUNTER_NEVER_DELETE=1
    export COUNTER_NEVER_SET_TO=1
    ```

2. Begin interacting with your locked down `counter`:

    ```bash
    { [ -f ~/.bashrc ] && source ~/.bashrc; } || { [ -f ~/.zshrc ] && source ~/.zshrc; }
    counter -h
    ```

### Using Counters Commonly

```bash
[q@localhost]~% counter -v
1.0.1
[q@localhost]~% counter -env
COUNTER_USE_FORCE=false
COUNTER_NEVER_ADD=false
COUNTER_NEVER_RESET=false
COUNTER_NEVER_DELETE=false
COUNTER_NEVER_SET_TO=false
COUNTER_NEVER_SUBTRACT=false
COUNTER_DIR=/tmp/.counters
COUNTER_QUANTITY=1
[q@localhost]~% counter -name subscribers 
Error: directory /tmp/.counters does not exist
[q@localhost]~% counter -name subscribers -F
0
[q@localhost]~% counter -name subscribers   
0
[q@localhost]~% counter -name subscribers -add
1
[q@localhost]~% counter -name subscribers -sub
0
[q@localhost]~% counter -name subscribers -set 20
20
[q@localhost]~% counter -name subscribers -reset 
will reset counter subscribers to 0 after you re-run with -yes
[q@localhost]~% counter -name subscribers -reset -yes
0
[q@localhost]~% counter -name subscribers            
0
[q@localhost]~% counter -name subscribers -delete
deleting counter subscribers (0) when you re-run with -yes
[q@localhost]~% counter -name subscribers -delete -yes
counter subscribers deleted
```

### Using Counter Overrides

```bash
[q@localhost]~% counter -env
COUNTER_NEVER_DELETE=false
COUNTER_NEVER_SET_TO=false
COUNTER_NEVER_SUBTRACT=false
COUNTER_DIR=/tmp/.counters
COUNTER_QUANTITY=1
COUNTER_USE_FORCE=false
COUNTER_NEVER_ADD=false
COUNTER_NEVER_RESET=false
[q@localhost]~% export COUNTER_NEVER_DELETE=1
[q@localhost]~% counter -name subscribers -delete -yes
Error: never delete enabled
[q@localhost]~% unset COUNTER_NEVER_DELETE
[q@localhost]~% counter -name subscribers -delete -yes
Error: remove /tmp/.counters/.named.d0f7111ea4066b9f7cd0f5dd.counter: no such file or directory
counter subscribers deleted
[q@localhost]~% counter -name subscribers             
0
[q@localhost]~% counter -name subscribers -delete -yes
Error: remove /tmp/.counters/.named.d0f7111ea4066b9f7cd0f5dd.counter: no such file or directory
counter subscribers deleted
[q@localhost]~% counter -name subscribers -add        
1
[q@localhost]~% counter -name subscribers -add
2
[q@localhost]~% counter -name subscribers -add
3
[q@localhost]~% counter -name subscribers -add
4
[q@localhost]~% counter -name subscribers     
4
[q@localhost]~% counter -name subscribers -delete -yes
counter subscribers deleted
[q@localhost]~% export COUNTER_QUANTITY=3
[q@localhost]~% counter -name subscribers -reset -yes
0
[q@localhost]~% counter -name subscribers -add       
3
[q@localhost]~% counter -name subscribers -add
6
[q@localhost]~% counter -name subscribers -add
9
[q@localhost]~% counter -name subscribers -sub
6
[q@localhost]~% counter -name subscribers -sub
3
[q@localhost]~% counter -name subscribers -reset -yes 
0
[q@localhost]~% unset COUNTER_QUANTITY
[q@localhost]~% export COUNTER_NEVER_ADD=1 
[q@localhost]~% counter -name subscribers 
0
[q@localhost]~% counter -name subscribers -add
0
[q@localhost]~% counter -name subscribers -sub
-1
[q@localhost]~% counter -name subscribers -add
-1
[q@localhost]~% counter -name subscribers -sub
-2
[q@localhost]~% unset COUNTER_NEVER_ADD 
[q@localhost]~% counter -name subscribers -add
-1
[q@localhost]~% counter -name subscribers -add
0
[q@localhost]~% export COUNTER_NEVER_SUBTRACT=1
[q@localhost]~% counter -name subscribers -add 
1
[q@localhost]~% counter -name subscribers -sub
1
[q@localhost]~% unset COUNTER_NEVER_SUBTRACT
[q@localhost]~% counter -name subscribers -sub
0
[q@localhost]~% export COUNTER_NEVER_RESET=1
[q@localhost]~% counter -name subscribers -reset 100
Error: reset operation is disabled by the environment variable
[q@localhost]~% unset COUNTER_NEVER_RESET
[q@localhost]~% counter -name subscribers -reset 100
will reset counter subscribers to 0 after you re-run with -yes
[q@localhost]~% counter -name subscribers -reset 100 -yes
will reset counter subscribers to 0 after you re-run with -yes
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

Process finished with the exit code 0 
```

### Benchmark Performance

```log
goos: linux
goarch: amd64
pkg: github.com/andreimerlescu/countable
cpu: Intel(R) Xeon(R) W-3245 CPU @ 3.20GHz
BenchmarkWriteCounter
BenchmarkWriteCounter-9              	  462278	      2263 ns/op
BenchmarkReadCounter
BenchmarkReadCounter-9               	  243709	      4764 ns/op
BenchmarkGenerateCounterFileName
BenchmarkGenerateCounterFileName-9   	 1956519	       613.6 ns/op
BenchmarkEnsureDir
BenchmarkEnsureDir-9                 	   67927	     17570 ns/op
BenchmarkResolveSymlink
BenchmarkResolveSymlink-9            	  180133	      6361 ns/op
PASS

Process finished with the exit code 0
```