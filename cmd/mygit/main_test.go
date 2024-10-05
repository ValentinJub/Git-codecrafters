package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"

	objects "github.com/codecrafters-io/git-starter-go/objects"
	util "github.com/codecrafters-io/git-starter-go/utils"
)

const (
	TEMPDIR  string = "/Users/valentin/temp/git/"
	TEMPDIR1 string = TEMPDIR + "git1"
	APP      string = "/tmp/codecrafters-build-git-go"
)

type expectedFile struct {
	Name  string
	IsDir bool
}

var ExpectedInitFiles = []expectedFile{
	{Name: "config", IsDir: false},
	{Name: "HEAD", IsDir: false},
	{Name: "hooks", IsDir: true},
	{Name: "objects", IsDir: true},
	{Name: "refs", IsDir: true},
}

var TestCaseHashObject = []struct {
	Description  string
	FileName     string
	Content      string
	ExpectedHash string
	ExpectedPath string
}{
	{
		Description:  "one file",
		FileName:     "test.txt",
		Content:      "hello world",
		ExpectedHash: "3b18e512dba79e4c8300dd08aeb37f8e728b8dad",
		ExpectedPath: ".git/objects/3b/18e512dba79e4c8300dd08aeb37f8e728b8dad",
	},

	{
		Description:  "some more text",
		FileName:     "prout.txt",
		Content:      "func helloWordl(s string) *Void {}",
		ExpectedHash: "0501091fcf64fe2b351473f1abb3a6fcb967ee93",
		ExpectedPath: ".git/objects/05/01091fcf64fe2b351473f1abb3a6fcb967ee93",
	},
}

var TestCaseCatFile = []struct {
	Description string
	ObjectHash  string
	Content     string
	Blob        objects.BlobObject
}{
	{
		Description: "simple hello world file",
		ObjectHash:  "3b18e512dba79e4c8300dd08aeb37f8e728b8dad",
		Content:     "blob 12\x00hello world",
		Blob: objects.BlobObject{
			ObjectHeader: objects.ObjectHeader{
				Type:   "blob",
				Length: "12", // len of content + 1 newline char "\n"
			},
			Content: "hello world",
		},
	},
	{
		Description: "more text",
		ObjectHash:  "0501091fcf64fe2b351473f1abb3a6fcb967ee93",
		Content:     "blob 34\x00func helloWordl(s string) *Void {}",
		Blob: objects.BlobObject{
			ObjectHeader: objects.ObjectHeader{
				Type:   "blob",
				Length: "35", // len of content + 1 newline char "\n"
			},
			Content: "func helloWordl(s string) *Void {}",
		},
	},
}

/*

ALL THE TESTS BELOW RELY ON THE PREVIOUS TEST'S SUCCESS, THE TESTS EXEC STOPS AT THE FIRST FAILURE

*/

// Test that the app creates a proper .git folder with the init command
func TestMyGit_Init(t *testing.T) {
	err := clearTempDir()
	util.Check(err)
	err = initRepo(TEMPDIR1)
	util.Check(err)
	err = os.Chdir(TEMPDIR1)
	util.Check(err)
	c, errDir := os.ReadDir(".git")
	util.Check(errDir)

	fileMap := make(map[string]bool)
	for _, entry := range c {
		fileMap[entry.Name()] = entry.IsDir()
	}

	for _, expectedFile := range ExpectedInitFiles {
		if isDir, found := fileMap[expectedFile.Name]; !found {
			log.Fatalf("Expected %s file to exist", expectedFile.Name)
		} else {
			if expectedFile.IsDir && !isDir {
				log.Fatalf("Expected %s to be a directory", expectedFile.Name)
			} else if !expectedFile.IsDir && isDir {
				log.Fatalf("Expected %s to be a file", expectedFile.Name)
			}
		}
	}
}

// Test that the app properly hashes objects, this one only tests for file hashing
func TestMyGit_HashObject(t *testing.T) {
	for _, tc := range TestCaseHashObject {
		t.Run(tc.Description, func(t *testing.T) {
			err := os.Chdir(TEMPDIR1)
			util.Check(err)
			err = util.Mkfile([]string{tc.FileName}, [][]byte{[]byte(tc.Content + "\n")}, 0644)
			util.Check(err)

			hash, errH := useHashObject(tc.FileName)
			util.Check(errH)

			if hash != tc.ExpectedHash {
				log.Fatalf(fmt.Sprintf("the expected hash differs from the hash returned\nGot:%s\nExp:%s\n", hash, tc.ExpectedHash))
			}
			if _, err := os.Open(tc.ExpectedPath); err != nil {
				log.Fatalf(fmt.Sprintf("the expected file doesn't exist at path %s", tc.ExpectedPath))
			} else {
				// Decode the file content (which also validates the file exists at the expected path)
				if blob, err := decodeBlobObject(tc.ExpectedPath); err != nil {
					util.Check(err)
				} else if blob.Content != tc.Content {
					log.Fatalf("\nd file content: %v\ne file content: %v", []byte(blob.Content), []byte(tc.Content))
				}
			}
		})
	}
}

// Test the app's ability to decode a git object file
// This test relies on the previous test's ability to write the expected git object file
func TestMyGit_CatFile(t *testing.T) {
	for _, tc := range TestCaseCatFile {
		t.Run(tc.Description, func(t *testing.T) {
			err := os.Chdir(TEMPDIR1)
			util.Check(err)
			for _, flag := range []string{"-t", "-s", "-p"} {
				out, err := useCatFile(tc.ObjectHash, flag)
				util.Check(err)
				switch flag {
				case "-t":
					if tc.Blob.Type != out {
						log.Fatalf("unexpected type returned, got: %s expected: %s", out, tc.Blob.Type)
					}
				case "-s":
					if tc.Blob.Length != out {
						log.Fatalf("unexpected length returned, got: %s expected: %s", out, tc.Blob.Length)
					}
				case "-p":
					if tc.Blob.Content != out {
						log.Fatalf("unexpected content returned, got: %s expected: %s", out, tc.Blob.Content)
					}
				}
			}
		})
	}
}

// Clears the tempdir where we create our test files and folders
func clearTempDir() error {
	if err := os.RemoveAll(TEMPDIR); err != nil {
		return fmt.Errorf("error deleting tempdir content: %s", err)
	}
	if err := os.MkdirAll(TEMPDIR, 0755); err != nil {
		return fmt.Errorf("error recreating the tempdir: %s", err)
	}
	return nil
}

// Use the app to use the cat file command on an object at path
func useCatFile(path, flag string) (string, error) {
	args := []string{"cat-file", flag, path}
	cmd := exec.Command(APP, args...)
	cmd.Dir = TEMPDIR1
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Use the app to use the hash object command on a file at path
func useHashObject(path string) (string, error) {
	args := []string{"hash-object", "-w", path}
	cmd := exec.Command(APP, args...)
	cmd.Dir = TEMPDIR1
	hash, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Use the app to init a new git repo in the given path
func initRepo(path string) error {
	if err := util.Mkdir(0755, path); err != nil {
		return err
	}
	args := []string{"init"}
	cmd := exec.Command(APP, args...)
	cmd.Dir = path
	_, err := cmd.Output()
	if err != nil {
		return err
	}
	return nil
}
