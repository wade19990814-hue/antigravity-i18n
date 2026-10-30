package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// VERSION is the single source of truth for the release number, and build.ps1
// copies it into wails.json because Wails fills the compiled binary's PE
// resource fields from there. If the two disagree, the executable's file
// version would contradict the version the app reports at runtime.
func TestVersionFileMatchesWailsConfig(t *testing.T) {
	raw, err := os.ReadFile("VERSION")
	if err != nil {
		t.Fatalf("reading VERSION: %v", err)
	}
	version := strings.TrimSpace(string(raw))
	if version == "" {
		t.Fatal("VERSION is empty")
	}

	cfg, err := os.ReadFile("wails.json")
	if err != nil {
		t.Fatalf("reading wails.json: %v", err)
	}
	var parsed struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(cfg, &parsed); err != nil {
		t.Fatalf("parsing wails.json: %v", err)
	}
	if parsed.Info.ProductVersion != version {
		t.Errorf("wails.json productVersion = %q, VERSION = %q; run build.ps1 or align them",
			parsed.Info.ProductVersion, version)
	}
}
