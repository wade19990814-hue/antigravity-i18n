package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"antigravity-i18n/internal/core"
)

// displayLocales lists the locale packs with coverage, for the CLI.
func displayLocales(svc *core.Service) []core.LocaleInfo {
	locales, err := svc.DescribeLocales()
	if err != nil {
		return nil
	}
	return locales
}

func fail(msg string) int {
	fmt.Fprintln(os.Stderr, "error:", msg)
	return 1
}

// confirmShutdown asks before the tool closes a running application. Patching
// app.asar requires the app to be closed, and doing that without asking would
// discard unsaved work.
func confirmShutdown(what string) bool {
	fmt.Println(what)
	fmt.Print("Antigravity may need to be closed. Continue? [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
