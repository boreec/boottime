package model

import (
	"errors"
	"strings"
	"time"
)

type BootTimeRecord struct {
	Firmware  time.Duration
	Loader    time.Duration
	Kernel    time.Duration
	Initrd    time.Duration
	Userspace time.Duration
	Total     time.Duration
}

func ParseSystemdAnalyzeTimeOutput(output string) (*BootTimeRecord, error) {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return nil, errors.New("empty output")
	}

	line := lines[0]
	words := strings.Fields(line)

	// Startup finished in 1.762s (firmware) + 265ms (loader) + 640ms (kernel) + 196ms (initrd) + 1.667s (userspace) = 4.532s
	var record BootTimeRecord
	var err error
	for idx, word := range words {
		switch {
		case strings.Contains(word, "(firmware)"):
			record.Firmware, err = time.ParseDuration(words[idx-1])
		case strings.Contains(word, "(loader)"):
			record.Loader, err = time.ParseDuration(words[idx-1])
		case strings.Contains(word, "(kernel)"):
			record.Kernel, err = time.ParseDuration(words[idx-1])
		case strings.Contains(word, "(initrd)"):
			record.Initrd, err = time.ParseDuration(words[idx-1])
		case strings.Contains(word, "(userspace)"):
			record.Userspace, err = time.ParseDuration(words[idx-1])
		case strings.Contains(word, "="):
			record.Total, err = time.ParseDuration(words[idx+1])
		}
		if err != nil {
			return nil, err
		}
	}
	return &record, nil
}
