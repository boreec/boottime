package exec

import (
	"fmt"
	"os/exec"

	"github.com/boreec/bootprobe/model"
)

func RunSystemdAnalyzeWithTime() (*model.SystemdAnalyzeTimeRecord, error) {
	cmd := exec.Command("systemd-analyze", "time")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("systemd-analyze failed: %w", err)
	}
	return model.ParseSystemdAnalyzeTimeOutput(string(out))
}
