package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// defaultRootFolderName is the default name for the root storage folder.
const defaultRootFolderName = "ggnetwork"

// CASPathTransformFunc implements a Content-Addressable Storage (CAS) path transformation.
// It creates a unique file path based on the SHA1 hash of the key.
func CASPathTransformFunc(key string) PathKey {
	// Generate SHA1 hash of the key
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	// Create a path structure by splitting the hash into blocks
	blocksize := 5
	sliceLen := len(hashStr) / blocksize
	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blocksize, (i*blocksize)+blocksize
		paths[i] = hashStr[from:to]
	}

	// Return the PathKey structure
	return PathKey{
		PathName: strings.Join(paths, "/"),
		Filename: hashStr,
	}
}

// PathTransformFunc is a type for functions that transform keys into file paths.
type PathTransformFunc func(string) PathKey

// PathKey represents a transformed file path and filename.
type PathKey struct {
	PathName string
	Filename string
}

// FirstPathName returns the first component of the path.
func (p PathKey) FirstPathName() string {
	paths := strings.Split(p.PathName, "/")
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

// FullPath returns the complete path including filename.
func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.PathName, p.Filename)
}

// StoreOpts contains options for configuring the Store.
type StoreOpts struct {
	Root              string
	PathTransformFunc PathTransformFunc
}

// DefaultPathTransformFunc is a simple path transform function that uses the key directly.
var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		PathName: key,
		Filename: key,
	}
}

// Store represents the file storage system.
type Store struct {
	StoreOpts
}

// NewStore creates a new Store with the given options.
func NewStore(opts StoreOpts) *Store {
	// Set default values if not provided
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}

	return &Store{
		StoreOpts: opts,
	}
}

// Has checks if a file exists in the store.
func (s *Store) Has(id string, key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	_, err := os.Stat(fullPathWithRoot)
	return !errors.Is(err, os.ErrNotExist)
}

// Clear removes all files from the store.
func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

// Delete removes a file from the store.
func (s *Store) Delete(id string, key string) error {
	pathKey := s.PathTransformFunc(key)

	defer func() {
		log.Printf("deleted [%s] from disk", pathKey.Filename)
	}()

	firstPathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FirstPathName())

	return os.RemoveAll(firstPathNameWithRoot)
}

// Write stores a file in the store.
func (s *Store) Write(id string, key string, r io.Reader) (int64, error) {
	return s.writeStream(id, key, r)
}

// WriteDecrypt stores an encrypted file in the store.
func (s *Store) WriteDecrypt(encKey []byte, id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}
	n, err := copyDecrypt(encKey, r, f)
	return int64(n), err
}

// openFileForWriting prepares a file for writing.
func (s *Store) openFileForWriting(id string, key string) (*os.File, error) {
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.PathName)
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	return os.Create(fullPathWithRoot)
}

// writeStream writes data from a reader to a file.
func (s *Store) writeStream(id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}
	return io.Copy(f, r)
}

// Read retrieves a file from the store.
func (s *Store) Read(id string, key string) (int64, io.Reader, error) {
	return s.readStream(id, key)
}

// readStream reads data from a file into a reader.
func (s *Store) readStream(id string, key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	file, err := os.Open(fullPathWithRoot)
	if err != nil {
		return 0, nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	return fi.Size(), file, nil
}
