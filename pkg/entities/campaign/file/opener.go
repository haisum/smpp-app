package file

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Opener is interface to read/write a file
type Opener interface {
	Open(filename string) (io.ReadWriteCloser, error)
}

// OpenReadWriteCloser is combination of Open + io.ReadWriteCloser
type OpenReadWriteCloser interface {
	Opener
	io.ReadWriteCloser
}

type opener struct {
	path      string
	filepath  string
	writeFile *os.File
	readFile  *os.File
}

// NewOpener returns a new instance of opener which satisfies OpenReadWriteCloser interface
func NewOpener(path string) *opener {
	return &opener{
		path: path,
	}
}

// Open saves filename as path+filename using path provided in NewOpener method.
// directory is created if not already present
func (o *opener) Open(filename string) (io.ReadWriteCloser, error) {
	err := os.MkdirAll(o.path, 0711)
	if err != nil {
		return nil, err
	}
	o.filepath = filepath.Join(o.path, filename)
	return o, nil
}

// Read opens file as defined in Open method and reads it
// this method satisfied io.Reader
func (o *opener) Read(p []byte) (n int, err error) {
	if o.readFile == nil {
		o.readFile, err = os.Open(o.filepath)
	}
	if err != nil {
		return 0, err
	}
	return o.readFile.Read(p)
}

// Write creates file as defined in Open method and writes to it
// this method satisfied io.Writer
func (o *opener) Write(p []byte) (n int, err error) {
	if o.writeFile == nil {
		o.writeFile, err = os.Create(o.filepath)
		err = o.validate(p)
		if err != nil {
			return 0, err
		}
	}
	return o.writeFile.Write(p)
}

// Close closes any open file pointers
// This satisfies io.Closer interface
func (o *opener) Close() error {
	if o.writeFile != nil {
		o.writeFile.Close()
	}
	if o.readFile != nil {
		o.readFile.Close()
	}
	o.writeFile = nil
	o.readFile = nil
	return nil
}

func (o *opener) validate(p []byte) error {
	if http.DetectContentType(p) != "text/plain; charset=utf-8" && http.DetectContentType(p) != "application/zip" {
		return errors.New("file doesn't seem to be a text or excel file")
	}
	return nil
}
