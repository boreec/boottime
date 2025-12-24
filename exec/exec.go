package exec

import (
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/boreec/boottime/model"

	"github.com/godbus/dbus/v5"
)

func RetrieveBootTime(useDbus bool) (*model.BootTimeRecord, error) {
	if useDbus {
		rec, err := RetrieveBootTimeWithDBUS()
		if err != nil {
			return nil, fmt.Errorf("retrieving with dbus: %w", err)
		}
		return rec, nil
	}

	rec, err := RetrieveBootTimeWithSystemdAnalyze()
	if err != nil {
		return nil, fmt.Errorf("retrieving with systemd-analyze: %w", err)
	}
	return rec, nil
}

func RetrieveBootTimeWithSystemdAnalyze() (*model.BootTimeRecord, error) {
	cmd := exec.Command("systemd-analyze", "time")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}
	return model.ParseSystemdAnalyzeTimeOutput(string(out))
}

func RetrieveBootTimeWithDBUS() (*model.BootTimeRecord, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	obj := conn.Object("org.freedesktop.systemd1", "/org/freedesktop/systemd1")

	var firmwareTs, loaderTs, kernelTs, initrdTs, userspaceTs, totalTs uint64
	properties := map[string]*uint64{
		"FirmwareTimestampMonotonic":  &firmwareTs,
		"LoaderTimestampMonotonic":    &loaderTs,
		"KernelTimestamp":             &kernelTs,
		"InitRDTimestampMonotonic":    &initrdTs,
		"UserspaceTimestampMonotonic": &userspaceTs,
		"FinishTimestampMonotonic":    &totalTs,
	}

	for propName, dest := range properties {
		var value dbus.Variant
		err := obj.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.systemd1.Manager", propName).Store(&value)
		if err != nil {
			// Some properties might not be available on all systems
			continue
		}

		if val, ok := value.Value().(uint64); ok {
			*dest = val
		}
	}

	if totalTs == 0 {
		return nil, errors.New("bootup is not yet finished")
	}

	usec := func(us uint64) time.Duration {
		return time.Duration(us) * time.Microsecond
	}
	// Determine kernel done time
	var kernelDoneTime uint64
	if initrdTs > 0 {
		kernelDoneTime = initrdTs
	} else {
		kernelDoneTime = userspaceTs
	}

	record := &model.BootTimeRecord{}

	if firmwareTs > 0 && loaderTs > 0 {
		record.Firmware = usec(firmwareTs - loaderTs)
	}

	if loaderTs > 0 {
		record.Loader = usec(loaderTs)
	}

	record.Kernel = usec(kernelDoneTime)

	if initrdTs > 0 {
		record.Initrd = usec(userspaceTs - initrdTs)
	}

	record.Userspace = usec(totalTs - userspaceTs)
	record.Total = usec(firmwareTs + totalTs)

	return record, nil
}
