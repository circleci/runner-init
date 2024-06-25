package main

import (
	"bytes"
	"testing"

	"github.com/alecthomas/kong"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestHelp(t *testing.T) {
	cli := &cli{}

	var tests = []struct {
		name string
		cli  any

		wantFilename string
	}{
		{
			name:         "check top-level help",
			cli:          cli,
			wantFilename: "help.txt",
		},
		{
			name:         "check init command help",
			cli:          &cli.Init,
			wantFilename: "init.txt",
		},
		{
			name:         "check run-task command help",
			cli:          &cli.RunTask,
			wantFilename: "run-task.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Check(t, golden.String(help(t, tt.cli), tt.wantFilename))
		})
	}
}

func help(t *testing.T, cli interface{}) string {
	t.Helper()

	w := bytes.NewBuffer(nil)
	rc := -1
	app, err := kong.New(cli,
		kong.Name("test-app"),
		kong.Writers(w, w),
		kong.Exit(func(i int) {
			rc = i
		}),
	)
	assert.Check(t, err)

	// Intentionally ignore the error, as it's not useful
	_, _ = app.Parse([]string{"--help"})
	assert.Check(t, cmp.Equal(0, rc))

	return w.String()
}
