package main

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocgen(t *testing.T) {
	d := NewDocgen()

	var out bytes.Buffer

	ctx := context.Background()
	err := d.Parse(ctx, "./testdata", &out)
	require.NoError(t, err)

	expected, err := os.ReadFile("testdata/doc.md")
	require.NoError(t, err)

	assert.Equal(t, string(expected), out.String())
}
