// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"testing"
)

func TestRootCommand(t *testing.T) {
	b := new(bytes.Buffer)
	RootCmd.SetOut(b)
	RootCmd.SetErr(b)
	RootCmd.SetArgs([]string{"version"})

	err := RootCmd.Execute()
	if err != nil {
		t.Fatalf("RootCmd execute failed: %v", err)
	}
}

func TestCapabilitiesCommand(t *testing.T) {
	b := new(bytes.Buffer)
	RootCmd.SetOut(b)
	RootCmd.SetErr(b)
	RootCmd.SetArgs([]string{"capabilities"})

	err := RootCmd.Execute()
	if err != nil {
		t.Fatalf("capabilities execution failed: %v", err)
	}
}

func TestConfigListCommand(t *testing.T) {
	b := new(bytes.Buffer)
	RootCmd.SetOut(b)
	RootCmd.SetErr(b)
	RootCmd.SetArgs([]string{"config", "list"})

	err := RootCmd.Execute()
	if err != nil {
		t.Fatalf("config list failed: %v", err)
	}
}
