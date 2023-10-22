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

		err := handlePatch(`testdata/sample.diff`, getWriter)
		require.NoError(t, err)

		assertMatchesFileContents(t, `testdata/test_with_diff_exp.go`, sb.String())
	})

	t.Run(`same file without patch`, func(t *testing.T) {
		sb := &strings.Builder{}

		err := writeFile(`testdata/test_with_diff.go`, allNodes, sb)
		require.NoError(t, err)

		assertMatchesFileContents(t, `testdata/test_without_diff_exp.go`, sb.String())
	})
}

func TestFormat(t *testing.T) {
	sb := &strings.Builder{}

	err := writeFile(`testdata/complex.go`, allNodes, sb)
	require.NoError(t, err)

	assertMatchesFileContents(t, `testdata/complex_exp.go`, sb.String())
}

func assertMatchesFileContents(t testing.TB, expFile string, actContents string) {
	t.Helper()

	bs, err := os.ReadFile(expFile)
	require.NoError(t, err)
	assert.Equal(t, string(bs), actContents)
}
