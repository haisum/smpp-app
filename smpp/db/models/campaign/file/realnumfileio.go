package file

import (
	"bitbucket.org/codefreak/hsmpp/smpp/stringutils"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

type RealNumFileIO struct {
	b []byte
}

func (nio *RealNumFileIO) Load(file io.Reader) ([]byte, error) {
	var err error
	nio.b, err = ioutil.ReadAll(file)
	if err != nil {
		return nio.b, errors.New("Couldn't read file.")
	}
	if http.DetectContentType(nio.b) != "text/plain; charset=utf-8" && http.DetectContentType(nio.b) != "application/zip" {
		return nio.b, errors.New("File doesn't seem to be a text or excel file.")
	}
	return nio.b, nil
}

func (nio *RealNumFileIO) LoadFile(filename string) ([]byte, error) {
	var err error
	nio.b, err = ioutil.ReadFile(filename)
	if err != nil {
		return nio.b, errors.New("couldn't read file")
	}
	if http.DetectContentType(nio.b) != "text/plain; charset=utf-8" && http.DetectContentType(nio.b) != "application/zip" {
		return nio.b, errors.New("file doesn't seem to be a text or excel file")
	}
	return nio.b, nil
}

func (nio *RealNumFileIO) Write(file *NumFile) error {
	if file.LocalName == "" {
		return errors.New("local name can't be blank")
	}
	file.LocalName = stringutils.SecureRandomAlphaString(20)
	numfilePath := filepath.Join(Path, file.Username)
	err := os.MkdirAll(numfilePath, 0711)
	if err != nil {
		return fmt.Errorf("couldn't create directory %s", numfilePath)
	}
	err = ioutil.WriteFile(filepath.Join(numfilePath, file.LocalName), nio.b, 0600)
	if err != nil {
		return fmt.Errorf("couldn't write file to disk at path %s", filepath.Join(numfilePath, file.LocalName))
	}
	return nil
}
