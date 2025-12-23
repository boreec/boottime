package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/boreec/bootprobe/exec"
	"github.com/boreec/bootprobe/model"
)

func main() {
	var flagTimeAnalysis, flagAverageAggregate bool
	var flagAggregateFileName string
	flag.BoolVar(&flagTimeAnalysis, "t", false, "run time analysis")
	flag.BoolVar(&flagAverageAggregate, "A", false, "print the average of a file aggregate")
	flag.StringVar(&flagAggregateFileName, "f", "", "concatenate systemd-analyse results to the file")
	flag.Parse()

	if flagTimeAnalysis && flagAverageAggregate {
		panic("time analysis and average can't be run together")
	}

	if !flagTimeAnalysis && !flagAverageAggregate {
		panic("no actions selected")
	}
	if flagTimeAnalysis && flagAggregateFileName == "" {
		panic("-f is required when using -t")
	}

	if flagAverageAggregate && flagAggregateFileName == "" {
		panic("-f is required when using -A")
	}

	var aggregateFile *os.File
	var err error
	if flagTimeAnalysis {
		aggregateFile, err = os.OpenFile(
			flagAggregateFileName,
			os.O_CREATE|os.O_APPEND|os.O_WRONLY,
			0o644,
		)
	} else {
		aggregateFile, err = os.Open(flagAggregateFileName)
	}
	if err != nil {
		panic(err.Error())
	}
	defer aggregateFile.Close()

	if flagTimeAnalysis {
		record, err := exec.RunSystemdAnalyzeWithTime()
		if err != nil {
			panic(err.Error())
		}

		enc := json.NewEncoder(aggregateFile)
		if err := enc.Encode(record); err != nil {
			panic(err.Error())
		}
	}

	if flagAverageAggregate {
		scanner := bufio.NewScanner(aggregateFile)
		var avg model.SystemdAnalyzeTimeRecord
		recCount := 0
		for scanner.Scan() {
			var rec model.SystemdAnalyzeTimeRecord
			if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
				panic(err)
			}
			avg.Firmware += rec.Firmware
			avg.Loader += rec.Loader
			avg.Kernel += rec.Kernel
			avg.Initrd += rec.Initrd
			avg.Userspace += rec.Userspace
			avg.Total += rec.Total
			recCount++
		}
		if recCount == 0 {
			panic("no records found")
		}

		avg.Firmware /= time.Duration(recCount)
		avg.Loader /= time.Duration(recCount)
		avg.Kernel /= time.Duration(recCount)
		avg.Initrd /= time.Duration(recCount)
		avg.Userspace /= time.Duration(recCount)
		avg.Total /= time.Duration(recCount)

		var b strings.Builder
		b.WriteString("average of ")
		b.WriteString(strconv.Itoa(recCount))
		b.WriteString(" records: ")

		b.WriteString(avg.Firmware.String())
		b.WriteString(" (firmware) + ")

		b.WriteString(avg.Loader.String())
		b.WriteString(" (loader) + ")

		b.WriteString(avg.Kernel.String())
		b.WriteString(" (kernel) + ")

		b.WriteString(avg.Initrd.String())
		b.WriteString(" (initrd) + ")

		b.WriteString(avg.Userspace.String())
		b.WriteString(" (userspace) = ")
		b.WriteString(avg.Total.String())
		b.WriteString("\n")

		io.WriteString(os.Stdout, b.String())
	}
}
