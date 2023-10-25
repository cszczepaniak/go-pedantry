package main

import (
	"os"
	"strings"
	"testing"

	"github.com/cszczepaniak/go-pedantry/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatch(t *testing.T) {
	t.Run(`with patch`, func(t *testing.T) {
		sb := &strings.Builder{}
		err := run(
			config.Config{
				Patch: `testdata/sample.diff`,
			},
			sb,
		)
		require.NoError(t, err)
		assertMatchesFileContents(t, `testdata/test_with_diff_exp.go`, sb.String())
	})

	t.Run(`same file without patch`, func(t *testing.T) {
		sb := &strings.Builder{}
		err := run(
			config.Config{
				Input: `testdata/test_with_diff.go`,
			},
			sb,
		)
		require.NoError(t, err)
		assertMatchesFileContents(t, `testdata/test_without_diff_exp.go`, sb.String())
	})
}

func TestFormat(t *testing.T) {
	sb := &strings.Builder{}
	err := run(
		config.Config{
			Input: `testdata/complex.go`,
		},
		sb,
	)
	require.NoError(t, err)
	assertMatchesFileContents(t, `testdata/complex_exp.go`, sb.String())
}

func TestList(t *testing.T) {
	sb := &strings.Builder{}
	err := run(
		config.Config{
			Input: `testdata`,
			List:  true,
		},
		sb,
	)
	require.NoError(t, err)
	assert.Equal(t, "testdata/complex.go\ntestdata/test_with_diff.go\ntestdata/test_with_diff_exp.go\n", sb.String())
}

func assertMatchesFileContents(t testing.TB, expFile string, actContents string) {
	t.Helper()

	bs, err := os.ReadFile(expFile)
	require.NoError(t, err)
	assert.Equal(t, string(bs), actContents)
}
