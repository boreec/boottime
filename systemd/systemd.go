package systemd

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

type BootTimeRecord struct {
	Firmware  time.Duration
	Loader    time.Duration
	Kernel    time.Duration
	Initrd    time.Duration
	Userspace time.Duration
	Total     time.Duration
}

func RetrieveBootTimeWithAnalyzeCommand() (*BootTimeRecord, error) {
	cmd := exec.Command("systemd-analyze", "time")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}
	return ParseSystemdAnalyzeTimeOutput(string(out))
}

func RetrieveBootTimeWithDbus() (*BootTimeRecord, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	obj := conn.Object("org.freedesktop.systemd1", "/org/freedesktop/systemd1")

	var firmwareTs, loaderTs, initrdTs, userspaceTs, finishTs uint64
	properties := map[string]*uint64{
		"FirmwareTimestampMonotonic":  &firmwareTs,
		"LoaderTimestampMonotonic":    &loaderTs,
		"InitRDTimestampMonotonic":    &initrdTs,
		"UserspaceTimestampMonotonic": &userspaceTs,
		"FinishTimestampMonotonic":    &finishTs,
	}

	for propName, dest := range properties {
		var value dbus.Variant
		err := obj.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.systemd1.Manager", propName).Store(&value)
		if err != nil {
			continue
		}

		if val, ok := value.Value().(uint64); ok {
			*dest = val
		}
	}

	if finishTs == 0 {
		return nil, errors.New("bootup is not yet finished")
	}

	usec := func(us uint64) time.Duration {
		return time.Duration(us) * time.Microsecond
	}

	// Determine kernel_done_time
	var kernelDoneTime uint64
	if initrdTs > 0 {
		kernelDoneTime = initrdTs
	} else {
		kernelDoneTime = userspaceTs
	}

	record := &BootTimeRecord{}

	// Match systemd's calculation exactly
	if firmwareTs > 0 && loaderTs > 0 {
		record.Firmware = usec(firmwareTs - loaderTs)
	}

	if loaderTs > 0 {
		record.Loader = usec(loaderTs)
	}

	record.Kernel = usec(kernelDoneTime)

	if initrdTs > 0 && userspaceTs > 0 {
		record.Initrd = usec(userspaceTs - initrdTs)
	}

	if finishTs > 0 && userspaceTs > 0 {
		record.Userspace = usec(finishTs - userspaceTs)
	}

	if firmwareTs > 0 && finishTs > 0 {
		record.Total = usec(firmwareTs + finishTs)
	}

	return record, nil
}

func ParseSystemdAnalyzeTimeOutput(output string) (*BootTimeRecord, error) {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return nil, errors.New("empty output")
	}

	line := lines[0]
	words := strings.Fields(line)

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
