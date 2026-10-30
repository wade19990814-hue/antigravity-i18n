package main

import (
	"fmt"
	"os"

	"antigravity-i18n/internal/core"
)

// One binary is both the windowed app and the command line tool. The locale
// store is built once and shared, so a window click and a console command can
// never disagree about which language packs exist.
func main() {
	store, err := core.NewLocaleStore(assetFS("assets"), assetFS("assets"),
		"locales", "patches")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	svc := core.NewService(store)
	if len(os.Args) > 1 {
		os.Exit(runCLI(svc, os.Args[1:]))
	}

	// The GUI is the default entry point; the CLI lives in the same binary.
	runWails(svc)
}
