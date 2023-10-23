package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testWriter struct {
	*strings.Builder
}

func (tw testWriter) Close() error {
	return nil
}

func newTestWriter() (func(string) (io.WriteCloser, error), *strings.Builder) {
	sb := &strings.Builder{}
	tw := testWriter{
		Builder: sb,
	}

	return func(s string) (io.WriteCloser, error) {
		return tw, nil
	}, sb
}

func TestPatch(t *testing.T) {
	t.Run(`with patch`, func(t *testing.T) {
		getWriter, sb := newTestWriter()

		listSink := &strings.Builder{}

		err := handlePatch(`testdata/sample.diff`, getWriter, listSink)
		require.NoError(t, err)

		assertMatchesFileContents(t, `testdata/test_with_diff_exp.go`, sb.String())
		assert.Equal(t, []string{`testdata/test_with_diff.go`}, strings.Split(strings.TrimSpace(listSink.String()), "\n"))
	})

	t.Run(`same file without patch`, func(t *testing.T) {
		listSink := &strings.Builder{}

		formatted, err := formatFile(`testdata/test_with_diff.go`, allNodes, listSink)
		require.NoError(t, err)

		assertMatchesFileContents(t, `testdata/test_without_diff_exp.go`, formatted)
		assert.Equal(t, []string{`testdata/test_with_diff.go`}, strings.Split(strings.TrimSpace(listSink.String()), "\n"))
	})
}

func TestFormat(t *testing.T) {
	listSink := &strings.Builder{}

	formatted, err := formatFile(`testdata/complex.go`, allNodes, listSink)
	require.NoError(t, err)

	assertMatchesFileContents(t, `testdata/complex_exp.go`, formatted)
	assert.Equal(t, []string{`testdata/complex.go`}, strings.Split(strings.TrimSpace(listSink.String()), "\n"))
}

func assertMatchesFileContents(t testing.TB, expFile string, actContents string) {
	t.Helper()

	bs, err := os.ReadFile(expFile)
	require.NoError(t, err)
	assert.Equal(t, string(bs), actContents)
}
