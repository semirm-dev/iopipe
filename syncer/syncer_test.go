package syncer_test

import (
	"context"
	"github.com/semirm-dev/iopipe/syncer"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSync(t *testing.T) {
	syncCtx, syncCancel := context.WithCancel(context.Background())
	defer syncCancel()

	steps := []syncer.Step{
		{
			ID:        "s1",
			InputDir:  "../tmp/input",
			OutputDir: "../tmp/output",
			Configs: map[string]string{
				"dir1": "01",
				"dir2": "01",
			},
		},
		{
			ID:        "s2",
			InputDir:  "../tmp/input",
			OutputDir: "../tmp/output",
			Configs: map[string]string{
				"dir1": "02/infile-1.3.txt",
			},
		},
	}

	err := syncer.Sync(syncCtx, steps)
	assert.NoError(t, err)
}
