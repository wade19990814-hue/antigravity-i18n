package core

import (
	"io"
	"os"
	"path/filepath"
)

// fileTxn snapshots auxiliary files an operation may touch. The live archive
// is committed separately with atomicReplaceFile, so rollback cannot overwrite
// an official update made by another process.
type fileTxn struct {
	baseDir string
	entries []txnEntry
}

type txnEntry struct {
	path    string
	existed bool
	data    []byte
	mode    os.FileMode
}

func newFileTxn() *fileTxn {
	return &fileTxn{}
}

// track records a file's current contents before it is modified. A file that
// does not exist yet is remembered as absent, so a rollback removes it.
func (t *fileTxn) track(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.entries = append(t.entries, txnEntry{path: path, existed: false})
			return nil
		}
		return err
	}
	if info.IsDir() {
		return &os.PathError{Op: "track", Path: path, Err: os.ErrInvalid}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	t.entries = append(t.entries, txnEntry{
		path:    path,
		existed: true,
		data:    data,
		mode:    info.Mode().Perm(),
	})
	return nil
}

// rollback restores every tracked file in reverse order, so the most recently
// written one is undone first. A file whose contents are already correct is
// skipped, which keeps a rollback from rewriting bytes it never touched.
func (t *fileTxn) rollback() []error {
	var errs []error
	for i := len(t.entries) - 1; i >= 0; i-- {
		e := t.entries[i]
		if !e.existed {
			if err := os.Remove(e.path); err != nil && !os.IsNotExist(err) {
				errs = append(errs, err)
			}
			continue
		}
		current, err := os.ReadFile(e.path)
		if err == nil && string(current) == string(e.data) {
			continue
		}
		if err := os.WriteFile(e.path, e.data, e.mode); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// failWithRollback wraps an error after undoing the operation, so the caller
// reports what actually failed rather than a rollback problem that followed it.
func (t *fileTxn) failWithRollback(what string, err error) error {
	err = explainWriteError(err)
	if rollbackErrs := t.rollback(); len(rollbackErrs) > 0 {
		return &RollbackError{Op: what, Cause: err, RollbackErrs: rollbackErrs}
	}
	return &OpError{Op: what, Cause: err}
}

// OpError names the step that failed.
type OpError struct {
	Op    string
	Cause error
}

func (e *OpError) Error() string {
	return e.Op + ": " + e.Cause.Error()
}

func (e *OpError) Unwrap() error { return e.Cause }

// RollbackError reports a failure whose cleanup also failed, which is the one
// case where the user may be left with a half-modified install.
type RollbackError struct {
	Op           string
	Cause        error
	RollbackErrs []error
}

func (e *RollbackError) Error() string {
	return e.Op + ": " + e.Cause.Error() + " (rollback also failed; the install may be inconsistent)"
}

func (e *RollbackError) Unwrap() error { return e.Cause }

// atomicReplaceFile moves tempPath over destPath through a temporary file in the
// destination directory, so the destination is never left half written. The
// optional verify callback runs while the staged file is still on disk next to
// the destination, letting a caller check both before committing.
func atomicReplaceFile(tempPath, destPath string, verify func(staged string) error) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	info, err := os.Stat(destPath)
	mode := os.FileMode(0o644)
	if err == nil {
		mode = info.Mode().Perm()
	}

	// Stage inside the destination directory so the rename never crosses a
	// filesystem boundary, where it would stop being atomic.
	staged, err := os.CreateTemp(filepath.Dir(destPath), ".antigravity-i18n-stage-*.asar")
	if err != nil {
		return err
	}
	stagedPath := staged.Name()
	defer os.Remove(stagedPath)

	in, err := os.Open(tempPath)
	if err != nil {
		staged.Close()
		return err
	}
	defer in.Close()
	if _, err := io.Copy(staged, in); err != nil {
		staged.Close()
		return err
	}
	if err := staged.Sync(); err != nil {
		staged.Close()
		return err
	}
	if err := staged.Close(); err != nil {
		return err
	}
	if err := os.Chmod(stagedPath, mode); err != nil {
		return err
	}

	if verify != nil {
		if err := verify(stagedPath); err != nil {
			return err
		}
	}

	// All writes and verification finish before replacing the destination.
	return os.Rename(stagedPath, destPath)
}

// copyFileExclusive copies src to dst, failing when dst already exists.
//
// Backups embed a timestamp accurate to the second, so two runs in the same
// second would otherwise silently overwrite one another's safety copy.
func copyFileExclusive(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return &os.PathError{Op: "copyfile", Path: dst, Err: os.ErrExist}
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
