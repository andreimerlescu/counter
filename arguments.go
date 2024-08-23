package main

import "flag"

type Argument struct {
	Short    [1]byte
	Long     string
	VarTo    interface{}
	Default  interface{}
	Usage    string
	Examples map[string]string // map[cmd]stdout = "counter <args>": "<stdout>"
}

var (
	useYes        bool
	showUsage     bool
	cycle         string
	cycleIn       string
	showEnv       bool
	doDeleteCycle bool
	doAdd         bool
	doSub         bool
	setTo         int64
	doReset       bool
	doDelete      bool
	showJson      bool
	useForce      bool
	quantity      int64
	showVersion   bool
	counterDir    string
	counterFile   string
	counterName   string
)

var args = []*Argument{
	{
		Short:   [1]byte{'n'},
		Long:    "name",
		VarTo:   &counterName,
		Default: DefaultCounterName,
		Usage:   "Counter name (alphanumeric only)",
		Examples: map[string]string{
			"counter -n test":                "0",
			"counter -name test -add":        "1",
			"counter -name test -sub":        "0",
			"counter -name test -set 7":      "7",
			"counter -name test -reset -yes": "0",
		},
	},
	{
		Short:   [1]byte{'d'},
		Long:    "dir",
		VarTo:   &counterDir,
		Default: DefaultCounterDir,
		Usage:   "Counter directory",
	},
	{
		Short:   [1]byte{'f'},
		Long:    "file",
		VarTo:   &counterFile,
		Default: DefaultCounterFile,
		Usage:   "Counter file name",
	},
	{
		Short:   [1]byte{'a'},
		Long:    "add",
		VarTo:   &doAdd,
		Default: DefaultDoAdd,
		Usage:   "Add -q=N (1) to the counter",
	},
	{
		Short:   [1]byte{'s'},
		Long:    "sub",
		VarTo:   &doSub,
		Default: DefaultDoSub,
		Usage:   "Subtract -q=N (1) from the counter",
	},
	{
		Short:   [1]byte{'S'},
		Long:    "set",
		VarTo:   &setTo,
		Default: DefaultSetTo,
		Usage:   "Set counter to value - 0 value ignores this flag",
	},
	{
		Short:   [1]byte{'R'},
		Long:    "reset",
		VarTo:   &doReset,
		Default: DefaultDoReset,
		Usage:   "Set counter to 0",
	},
	{
		Short:   [1]byte{'D'},
		Long:    "delete",
		VarTo:   &doDelete,
		Default: DefaultDoDelete,
		Usage:   "Delete the counter",
	},
	{
		Short:   [1]byte{'j'},
		Long:    "json",
		VarTo:   &showJson,
		Default: DefaultShowJson,
		Usage:   "Show JSON formatted output",
	},
	{
		Short:   [1]byte{'F'},
		Long:    "force",
		VarTo:   &useForce,
		Default: DefaultUseForce,
		Usage:   "Force overwrite",
	},
	{
		Short:   [1]byte{'q'},
		Long:    "quantity",
		VarTo:   &quantity,
		Default: DefaultQuantity,
		Usage:   "Quantity to either add/subtract from counter",
	},
	{
		Short:   [1]byte{'v'},
		Long:    "version",
		VarTo:   &showVersion,
		Default: DefaultShowVersion,
		Usage:   "Show version",
	},
	{
		Short:   [1]byte{'c'},
		Long:    "cycle",
		VarTo:   &cycle,
		Default: DefaultCycle,
		Usage:   "Time cycle for automatic counter reset (e.g., hourly, daily, weekly, etc.)",
	},
	{
		Short:   [1]byte{'i'},
		Long:    "in",
		VarTo:   &cycleIn,
		Default: DefaultCycleIn,
		Usage:   "Specific time within the cycle (e.g., 'noon', 'monday', '12:00')",
	},
	{
		Short:   [1]byte{'e'},
		Long:    "env",
		VarTo:   &showEnv,
		Default: DefaultShowEnv,
		Usage:   "Show environment variables",
	},
	{
		Short:   [1]byte{'y'},
		Long:    "yes",
		VarTo:   &useYes,
		Default: DefaultUseYes,
		Usage:   "Your response is yes",
	},
	{
		Short:   [1]byte{'r'},
		Long:    "rmcc",
		VarTo:   &doDeleteCycle,
		Default: DefaultDoDeleteCycle,
		Usage:   "Delete the cycle on the counter",
	},
}

func initArgs() {
	for _, arg := range args {
		switch v := arg.Default.(type) {
		case string:
			flag.StringVar(arg.VarTo.(*string), string(arg.Short[:]), v, arg.Usage)
			if len(arg.Long) > 0 {
				flag.StringVar(arg.VarTo.(*string), arg.Long, v, arg.Usage)
			}
		case int64:
			flag.Int64Var(arg.VarTo.(*int64), string(arg.Short[:]), v, arg.Usage)
			if len(arg.Long) > 0 {
				flag.Int64Var(arg.VarTo.(*int64), arg.Long, v, arg.Usage)
			}
		case bool:
			flag.BoolVar(arg.VarTo.(*bool), string(arg.Short[:]), v, arg.Usage)
			if len(arg.Long) > 0 {
				flag.BoolVar(arg.VarTo.(*bool), arg.Long, v, arg.Usage)
			}
		}
	}
	flag.Parse()
}
