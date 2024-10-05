package utils

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
)

// Check the content of an error, logfatalf if error is found
func Check(err error) {
	if err != nil {
		log.Fatalf(err.Error())
	}
}

// Create files at paths with contents
func Mkfile(paths []string, contents [][]byte, perm fs.FileMode) error {
	if len(paths) != len(contents) {
		return fmt.Errorf("paths and content aren't the same length")
	}
	for i, p := range paths {
		if err := os.WriteFile(p, contents[i], perm); err != nil {
			return err
		}
	}
	return nil
}

// Create dirs from paths
func Mkdir(perm fs.FileMode, paths ...string) error {
	for _, p := range paths {
		if err := os.MkdirAll(p, perm); err != nil {
			return err
		}
	}
	return nil
}

func ReadFile(file string) (*bytes.Buffer, error) {
	fileHandle, err := os.Open(file)
	if err != nil {
		return new(bytes.Buffer), fmt.Errorf("unable to open %s\nError: %s", file, err)
	}
	// Put the file data in a buffer we can read from
	b := new(bytes.Buffer)
	_, err = io.Copy(b, fileHandle)
	if err != nil {
		return new(bytes.Buffer), fmt.Errorf("error while reading from the file: %s", err)
	}
	return b, nil
}
