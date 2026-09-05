package reftable

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileWriterClosePreservesNewOwnerLock(t *testing.T) {
	storage := NewLocalStorage(t.TempDir())
	first, err := storage.LockForWrite(listFileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if first.Committed() {
		t.Fatal("aborted writer reports committed")
	}
	second, err := storage.LockForWrite(listFileName)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	third, err := storage.LockForWrite(listFileName)
	if err == nil {
		third.Close()
		t.Fatal("repeated Close removed another writer's lock")
	}
	if !os.IsExist(err) {
		t.Fatalf("got %v, want lock contention", err)
	}
}

func TestFileWriterFailedRenameCanBeCleaned(t *testing.T) {
	dir := t.TempDir()
	// Replacing a directory with a regular file must fail.
	dest := filepath.Join(dir, "destination")
	if err := os.Mkdir(dest, 0700); err != nil {
		t.Fatal(err)
	}
	w, err := newAtomicWriter(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	temp := filepath.Join(dir, w.Name())
	if err := w.Commit(); err == nil {
		t.Fatal("expected rename failure")
	}
	if w.Committed() {
		t.Fatal("failed rename reports committed")
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(temp); !os.IsNotExist(err) {
		t.Fatalf("temporary file survived cleanup: %v", err)
	}
}

func TestFileWriterCommittedFileSurvivesClose(t *testing.T) {
	dir := t.TempDir()
	w, err := newAtomicWriter(filepath.Join(dir, "table.ref"))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if _, err := w.Write([]byte("published")); err != nil {
		t.Fatal(err)
	}
	if err := w.Commit(); err != nil {
		t.Fatal(err)
	}
	if !w.Committed() || w.Name() != "table.ref" {
		t.Fatal("incorrect state after commit")
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "table.ref"))
	if err != nil || string(data) != "published" {
		t.Fatalf("published file changed after Close: %q, %v", data, err)
	}
}
