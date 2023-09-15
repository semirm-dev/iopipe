package syncer_test

import (
	"context"
	"github.com/semirm-dev/iopipe/syncer"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSync(t *testing.T) {
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

	err := syncer.Sync(context.Background(), steps)
	assert.NoError(t, err)
}
