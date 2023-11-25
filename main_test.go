package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildApp(t *testing.T) {
	require.Nil(t, app().Validate())
}
