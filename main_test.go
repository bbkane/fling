package main

import (
	"testing"
)

func TestBuildApp(t *testing.T) {
	if err := app().Validate(); err != nil {
		t.Fatal(err)
	}
}
