package main

import (
	"errors"
	"flag"
	"strings"

	"github.com/boreec/boottime/exec"
)

func main() {
	var args Args
	var flags Flags

	if err := parseArgs(&args, &flags); err != nil {
		panic(err.Error())
	}

	if err := runWithArgs(&args, &flags); err != nil {
		panic(err.Error())
	}
}

type Flags struct {
	RunRetrieveBootTime bool
	RunAggregate        bool
	Prettify            bool
}

type Args struct {
	FileName string
}

func parseArgs(args *Args, flags *Flags) error {
	flag.BoolVar(&flags.RunRetrieveBootTime, "R", false, "retrieve boot time")
	flag.BoolVar(&flags.RunRetrieveBootTime, "retrieve-boot-time", false, "retrieve boot time")

	flag.BoolVar(&flags.RunAggregate, "A", false, "average boot time records")
	flag.BoolVar(&flags.RunAggregate, "average-boot-records", false, "average boot time records")

	flag.BoolVar(&flags.Prettify, "p", false, "prettify results")
	flag.BoolVar(&flags.Prettify, "prettify", false, "prettify results")
	flag.Parse()

	argsUnparsed := flag.Args()
	if len(argsUnparsed) == 0 {
		return errors.New("expected 1 arg for jsonl file, found 0")
	}
	args.FileName = argsUnparsed[0]

	if !strings.HasSuffix(args.FileName, ".jsonl") {
		return errors.New("argument should be a file name with .jsonl suffix")
	}

	if flags.RunAggregate && flags.RunRetrieveBootTime {
		return errors.New("flags -A and -R are incompatible")
	}

	if !flags.RunAggregate && !flags.RunRetrieveBootTime {
		return errors.New("flags -A or -R required")
	}

	return nil
}

func runWithArgs(args *Args, flags *Flags) error {
	if flags.RunRetrieveBootTime {
		return exec.RetrieveBootTimes(args.FileName)
	}

	if flags.RunAggregate {
		return exec.PrintRecordsAverage(args.FileName, flags.Prettify)
	}

	return nil
}
