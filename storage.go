package reftable

import (
	"cmp"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// AtomicWriter is an abstraction for a {write to temp, close, rename}
// file sink.
type AtomicWriter interface {
	io.WriteCloser
	Name() string
	Commit() error
	Committed() bool
}

type fileWriter struct {
	finalName string
	*os.File
}

func (fw *fileWriter) Committed() bool {
	return fw.File == nil
}

func (fw *fileWriter) Name() string {
	if fw.File != nil {
		return filepath.Base(fw.File.Name())
	}

	return filepath.Base(fw.finalName)
}

func (fw *fileWriter) Close() error {
	if fw.File == nil {
		return nil
	}
	err1 := fw.File.Close()
	err2 := os.Remove(fw.File.Name())
	return cmp.Or(err1, err2)
}

func (fw *fileWriter) Commit() error {
	if err := fw.File.Sync(); err != nil {
		return err
	}
	if err := fw.File.Close(); err != nil {
		return err
	}

	err := os.Rename(fw.File.Name(), fw.finalName)
	fw.File = nil
	return err
}

func newLockForWrite(path string) (AtomicWriter, error) {
	f, err := os.OpenFile(path+".lock", os.O_EXCL|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &fileWriter{File: f, finalName: path}, nil
}

func newAtomicWriter(path string) (AtomicWriter, error) {
	name := filepath.Base(path)
	f, err := os.CreateTemp(filepath.Dir(path), name+".*.tmp")
	if err != nil {
		return nil, err
	}
	return &fileWriter{File: f, finalName: path}, nil
}

func (s *storage) LockForWrite(name string) (AtomicWriter, error) {
	if strings.HasPrefix(name, s.reftableDir) {
		log.Panicf("prefix")
	}

	return newLockForWrite(filepath.Join(s.reftableDir, name))
}

type storage struct {
	reftableDir string
}

func (s *storage) NewWriter(name string) (AtomicWriter, error) {
	if strings.HasPrefix(name, s.reftableDir) {
		log.Panicf("prefix")
	}
	return newAtomicWriter(filepath.Join(s.reftableDir, name))
}

func (s *storage) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.reftableDir, name))
}

func (s *storage) ReadDir() ([]fs.DirEntry, error) {
	return os.ReadDir(s.reftableDir)
}

func (s *storage) Sync() error {
	return fsyncDir(s.reftableDir)
}

func (s *storage) Remove(name string) error {
	return os.Remove(filepath.Join(s.reftableDir, name))
}

type fileBlockSource struct {
	f interface {
		io.ReaderAt
		io.Closer
	}
	sz uint64
}

// OpenBlockSource opens a file on local disk as a BlockSource.
func (s *storage) OpenBlockSource(name string) (BlockSource, error) {
	if strings.HasPrefix(name, s.reftableDir) {
		log.Panicf("prefix")
	}
	f, err := os.Open(filepath.Join(s.reftableDir, name))
	if err != nil {
		return nil, fmt.Errorf("NewFileBlockSource: %w", err)
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	return &fileBlockSource{f, uint64(fi.Size())}, nil
}

func (bs *fileBlockSource) Size() uint64 {
	return bs.sz
}

func (bs *fileBlockSource) ReadBlock(off uint64, size int) ([]byte, error) {
	if off >= bs.sz {
		return nil, io.EOF
	}
	if off+uint64(size) > bs.sz {
		size = int(bs.sz - off)
	}
	b := make([]byte, size)
	n, err := bs.f.ReadAt(b, int64(off))
	if err != nil {
		return nil, err
	}

	return b[:n], nil
}

func (bs *fileBlockSource) Close() error {
	return bs.f.Close()
}
