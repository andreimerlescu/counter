package main

import (
	"fmt"
	"os"
)

type Option struct {
	ShortFlag   string
	LongFlag    string
	Description string
}

type Example struct {
	Description string
	Command     string
}

var options = []Option{
	{"-env", "", "Show environment variables"},
	{"-usage", "", "Show this usage help"},
	{"-v", "-version", "Show current version"},
	{"-h", "-help", "Show flag usage help"},
	{"-n=", "-name <name>", "Name of the counter"},
	{"-d=", "-dir <name>", "Directory* to save counters"},
	{"-f=", "-file <name>", "Counter file path* to file to use as counter"},
	{"-a", "-add <int64>", "Add -q=N (1) to the counter"},
	{"-s", "-sub <int64>", "Subtract -q=N (1) from the counter"},
	{"-S=", "-set <int64>", "Set the counter to value -S=0 ignores this flag"},
	{"-R", "-reset", "Reset the counter to 0"},
	{"-D", "-delete", "Delete the counter"},
	{"-yes", "", "Confirm destructive actions on counters"},
	{"-cycle", "-cycle <type>", "Time cycle for automatic counter reset (e.g., hourly, daily, weekly, etc.)"},
	{"-in", "-in <time>", "Specific time within the cycle (e.g., 'noon', 'monday', '12:00')"},
	{"-json", "", "Show json formatted output"},
}

var examples = []Example{
	{"Basic Usage", "counter -name subscriptions -add"},
	{"Set a counter to 1000", "counter -name subscriptions -set 1000"},
	{"Reset a counter", "counter -name subscriptions -reset"},
	{"Use daily cycle", "counter -name daily_hits -cycle daily -in midnight"},
}

func printUsage() {
	fmt.Printf("Usage of counter: %s\n", os.Args[0])
	fmt.Println("")

	fmt.Println("+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
	fmt.Println("+                        [OPTIONS] Globally parsed                                  +")
	fmt.Println("+-----------------------------------------------------------------------------------+")

	// Determine the max lengths for dynamic padding
	shortFlagMaxLen := 0
	longFlagMaxLen := 0
	for _, opt := range options {
		if len(opt.ShortFlag) > shortFlagMaxLen {
			shortFlagMaxLen = len(opt.ShortFlag)
		}
		if len(opt.LongFlag) > longFlagMaxLen {
			longFlagMaxLen = len(opt.LongFlag)
		}
	}

	// Print the dynamically generated table
	for _, opt := range options {
		shortFlag := fmt.Sprintf("%-*s", shortFlagMaxLen, opt.ShortFlag)
		longFlag := fmt.Sprintf("%-*s", longFlagMaxLen, opt.LongFlag)
		fmt.Printf("| %s | %s | %s |\n", shortFlag, longFlag, opt.Description)
	}
	fmt.Println("-------------------------------------------------------------------------------------")

	fmt.Println("+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
	fmt.Println("+                            REAL WORLD EXAMPLE USAGE                               +")
	fmt.Println("+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")

	for _, example := range examples {
		fmt.Printf("| %s |\n", example.Command)
	}
	fmt.Println("-------------------------------------------------------------------------------------")

	fmt.Println("+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
	fmt.Println("+                       EXAMPLE OF BASIC DAILY CYCLE                                +")
	fmt.Println("+-----------------------------------------------------------------------------------+")
	fmt.Println("| counter -name daily_hits -cycle daily -in midnight                                |")
	fmt.Println("| Resets the counter 'daily_hits' every midnight                                    |")
	fmt.Println("-------------------------------------------------------------------------------------")

	fmt.Println("")
}
