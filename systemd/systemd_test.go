package systemd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAnalyzeCommandOutput(t *testing.T) {
	tcs := map[string]struct {
		input    string
		validate func(t *testing.T, btr *BootTimeRecord, err error, name string)
	}{
		"parse valid input successfully": {
			input: `Startup finished in 1.897s (firmware) + 1.715s (loader) + 718ms (kernel) + 2.049s (initrd) + 13.275s (userspace) = 19.656s
graphical.target reached after 13.270s in userspace.`,
			validate: func(t *testing.T, btr *BootTimeRecord, err error, name string) {
				require.NoError(t, err, name)
				require.NotNil(t, btr, name)
				assert.Equal(t, time.Duration(1897)*time.Millisecond, btr.Firmware, name)
				assert.Equal(t, time.Duration(1715)*time.Millisecond, btr.Loader, name)
				assert.Equal(t, time.Duration(718)*time.Millisecond, btr.Kernel, name)
				assert.Equal(t, time.Duration(2049)*time.Millisecond, btr.Initrd, name)
				assert.Equal(t, time.Duration(13275)*time.Millisecond, btr.Userspace, name)
				assert.Equal(t, time.Duration(19656)*time.Millisecond, btr.Total, name)
			},
		},
		"parse empty input returns error": {
			input: "",
			validate: func(t *testing.T, btr *BootTimeRecord, err error, name string) {
				require.ErrorIs(t, err, ErrParseAnalyzeCommandEmptyOutput, name)
				require.Nil(t, btr, name)
			},
		},
		"parse input with bad duration returns error": {
			input: `Startup finished in potatoes (firmware) + potatoes (loader) + potatoesms (kernel) + 2.049potatoes (initrd) + 13.275s (userspace) = 19.656s
graphical.target reached after 13.270s in userspace.`,
			validate: func(t *testing.T, btr *BootTimeRecord, err error, name string) {
				require.Error(t, err, name)
				require.Nil(t, btr, name)
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			btr, err := ParseAnalyzeCommandOutput(tc.input)
			tc.validate(t, btr, err, name)
		})
	}
}
