package main

import (
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func installBinary(t testing.TB) {
	t.Helper()

	sync.OnceFunc(func() {
		_, err := exec.Command(`go`, `install`).CombinedOutput()
		require.NoError(t, err)
	})()
}

func runCmd(t testing.TB, name string, args ...string) string {
	installBinary(t)

	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	cmd := exec.Command(name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	require.NoError(t, err, stderr.String())

	return stdout.String()
}

func TestPatch(t *testing.T) {
	t.Run(`with patch`, func(t *testing.T) {
		out := runCmd(t, `go-pedantry`, `-patch`, `testdata/sample.diff`)
		assertMatchesFileContents(t, `testdata/test_with_diff_exp.go`, out)
	})

	t.Run(`same file without patch`, func(t *testing.T) {
		out := runCmd(t, `go-pedantry`, `-input`, `testdata/test_with_diff.go`)
		assertMatchesFileContents(t, `testdata/test_without_diff_exp.go`, out)
	})
}

func TestFormat(t *testing.T) {
	out := runCmd(t, `go-pedantry`, `-input`, `testdata/complex.go`)
	assertMatchesFileContents(t, `testdata/complex_exp.go`, out)
}

func TestList(t *testing.T) {
	out := runCmd(t, `go-pedantry`, `-input`, `testdata`, `-l`)
	assert.Equal(t, "testdata/complex.go\ntestdata/test_with_diff.go\ntestdata/test_with_diff_exp.go\n", out)
}

func assertMatchesFileContents(t testing.TB, expFile string, actContents string) {
	t.Helper()

	bs, err := os.ReadFile(expFile)
	require.NoError(t, err)
	assert.Equal(t, string(bs), actContents)
}
