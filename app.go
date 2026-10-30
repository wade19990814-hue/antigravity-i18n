package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"antigravity-i18n/internal/core"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// shutdownWarning is what the confirmation dialog tells the user before either
// operation closes Antigravity. It is defined once because both apply and
// restore reach the same forced kill, so the warning cannot drift between them.
const shutdownWarning = "需要先关闭 Antigravity。\n\n" +
	"工具会请求正常退出并等待 30 秒，超时将强制结束进程。" +
	"未保存的编辑器内容、正在进行的会话或撤销历史可能丢失。\n\n是否继续？"

// App binds the Go core to the Wails frontend. The GUI only reaches the core
// through this type, so a window click and a CLI command perform the same work.
type App struct {
	ctx      context.Context
	svc      *core.Service
	appDir   string
	mu       sync.Mutex
	logs     []string
	progress func(stage, detail string)
}

// NewApp builds the GUI application around the shared locale store.
func NewApp(svc *core.Service, initialDir string) *App {
	return &App{svc: svc, appDir: initialDir}
}

// startup captures the context the runtime needs to emit events.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.progress = func(stage, detail string) {
		wailsRuntime.EventsEmit(ctx, "progress", map[string]string{
			"stage":  stage,
			"detail": detail,
		})
	}
	// The sink is installed once here rather than at each call site, so an
	// action taken before startup has already been set cannot go silent.
	a.svc.Progress = a.progress
	a.svc.Progress("就绪", "正在检测 Antigravity")
}

func (a *App) appendLog(msg string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.logs = append(a.logs, msg)
	if len(a.logs) > 400 {
		a.logs = a.logs[len(a.logs)-400:]
	}
}

func (a *App) currentDir() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.appDir
}

// GetVersion reports the tool version shown in the window chrome.
func (a *App) GetVersion() string { return core.Version }

// GetStatus reports the localization state of the selected installation.
func (a *App) GetStatus() (*core.Status, error) {
	st, err := a.svc.GetStatus(a.currentDir())
	if err != nil {
		return nil, err
	}
	if st != nil && st.AppDir != "" {
		a.mu.Lock()
		a.appDir = st.AppDir
		a.mu.Unlock()
	}
	return st, nil
}

// GetLocales lists every bundled language pack.
func (a *App) GetLocales() ([]core.LocaleInfo, error) {
	return a.svc.DescribeLocales()
}

// SelectAppDir opens a native folder picker.
func (a *App) SelectAppDir() (string, error) {
	if a.ctx == nil {
		return "", errors.New("app context not initialized")
	}
	selected, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select the Antigravity installation folder (the one holding Antigravity.exe)",
	})
	if err != nil {
		return "", err
	}
	return selected, nil
}

// SetAppDir targets another installation and returns the refreshed status.
func (a *App) SetAppDir(path string) (*core.Status, error) {
	if path == "" {
		return nil, errors.New("no directory selected")
	}
	st, err := a.svc.GetStatus(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.appDir = st.AppDir
	a.mu.Unlock()
	return st, nil
}

// ApplyLocale installs a language pack, replacing whatever patch is present.
//
// confirmed must be true only after the user accepted shutdownWarning. The
// window used to close Antigravity with no prompt at all, which is how unsaved
// work got discarded, so the refusal is enforced here rather than in the UI.
func (a *App) ApplyLocale(locale string, confirmed bool) (*core.ApplyResult, error) {
	if !confirmed {
		return nil, errors.New(shutdownWarning)
	}
	result, err := a.svc.ApplyLocale(core.Options{
		AppDir:    a.currentDir(),
		Locale:    locale,
		Restart:   true,
		TimeoutMs: 30000,
	})
	for _, note := range a.svc.Notes() {
		a.appendLog("note: " + note)
	}
	if err != nil {
		a.appendLog("error: " + err.Error())
		return nil, err
	}
	a.appendLog(fmt.Sprintf("applied %s (%s)", result.LocaleName, result.Locale))
	return result, nil
}

// RestoreLocale puts the official English build back. Restore stops the process
// exactly like apply does, so it demands the same confirmation.
func (a *App) RestoreLocale(confirmed bool) (*core.RestoreResult, error) {
	if !confirmed {
		return nil, errors.New(shutdownWarning)
	}
	result, err := a.svc.RestoreOfficial(core.Options{
		AppDir:    a.currentDir(),
		Restart:   true,
		TimeoutMs: 30000,
	})
	for _, note := range a.svc.Notes() {
		a.appendLog("note: " + note)
	}
	if err != nil {
		a.appendLog("error: " + err.Error())
		return nil, err
	}
	a.appendLog("restored the official English build")
	return result, nil
}

// LaunchAntigravity starts the application after a successful patch.
func (a *App) LaunchAntigravity() error {
	dir := a.currentDir()
	if dir == "" {
		return errors.New("no Antigravity installation selected")
	}
	exe := filepath.Join(dir, "Antigravity.exe")
	if _, err := os.Stat(exe); err != nil {
		return fmt.Errorf("Antigravity.exe was not found in %s", dir)
	}
	cmd := exec.Command(exe)
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()
	return nil
}

// GetLogs returns the diagnostics collected in this session.
func (a *App) GetLogs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, len(a.logs))
	copy(out, a.logs)
	return out
}
