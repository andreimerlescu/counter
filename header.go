package main

const (
	VERSION                 string = "2.0.0"
	DefaultCounterFile      string = ""
	DefaultCounterName      string = ""
	DefaultCounterDir       string = "/tmp/.counters"
	DefaultQuantity         int64  = 1
	DefaultSetTo            int64  = 0
	DefaultShowVersion      bool   = false
	DefaultShowUsage        bool   = false
	DefaultDoAdd            bool   = false
	DefaultDoSub            bool   = false
	DefaultDoDelete         bool   = false
	DefaultDoReset          bool   = false
	DefaultUseForce         bool   = false
	DefaultUseYes           bool   = false
	DefaultShowEnv          bool   = false
	DefaultNeverDelete      bool   = false
	DefaultNeverSubtract    bool   = false
	DefaultNeverReset       bool   = false
	DefaultNeverAdd         bool   = false
	DefaultNeverSetTo       bool   = false
	DefaultNeverCycle       bool   = false
	DefaultCycle            string = ""
	DefaultCycleIn          string = ""
	DefaultDoDeleteCycle    bool   = false
	DefaultNeverDeleteCycle bool   = false
	DefaultShowJson         bool   = false
)

var CounterEnv = map[string]interface{}{
	"COUNTER_DIR":                &counterDir,
	"COUNTER_QUANTITY":           &quantity,
	"COUNTER_USE_FORCE":          &useForce,
	"COUNTER_NEVER_ADD":          &neverAdd,
	"COUNTER_ALWAYS_YES":         &useYes,
	"COUNTER_NEVER_RESET":        &neverReset,
	"COUNTER_NEVER_DELETE":       &neverDelete,
	"COUNTER_NEVER_SET_TO":       &neverSetTo,
	"COUNTER_NEVER_SUBTRACT":     &neverSubtract,
	"COUNTER_NEVER_CYCLE":        &neverCycle,
	"COUNTER_NEVER_DELETE_CYCLE": &neverDeleteCycle,
}

var (
	neverDelete      bool = DefaultNeverDelete
	neverSubtract    bool = DefaultNeverSubtract
	neverReset       bool = DefaultNeverReset
	neverAdd         bool = DefaultNeverAdd
	neverSetTo       bool = DefaultNeverSetTo
	neverCycle       bool = DefaultNeverCycle
	neverDeleteCycle bool = DefaultNeverDeleteCycle
)
