package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"regexp"
)

type GitObject struct {
	Type    string
	Length  string
	Content string
}

// Usage: your_program.sh <command> <arg1> <arg2> ...
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
		// Initialize a new git repository, creating the necessary directories and files
		for _, dir := range []string{".git", ".git/objects", ".git/refs"} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			}
		}
		headFileContents := []byte("ref: refs/heads/main\n")
		if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
		}
		fmt.Println("Initialized git directory")
	case "cat-file":
		// Display information about .git/objects
		res, err := catFile(os.Args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while doing catfile stuff %s\n", err)
			os.Exit(1)
		}
		fmt.Print(res)
	default:
		// Undefined command
		fmt.Fprintf(os.Stderr, "Undefined command %s\n", command)
		os.Exit(1)
	}
}

// Display content, size or type of a git/object
// The content is encoded using zlib
// Example: mygit cat-file -p 4csejhtq23098ughaohjg
func catFile(args []string) (string, error) {
	if len(args) < 4 {
		return "", fmt.Errorf("usage: mygit cat-file <flags> <objects>\n")
	}

	flag := args[2]
	file := args[3]
	dir := string(file[:2])
	object := string(file[2:])
	filePath := fmt.Sprintf(".git/objects/%s/%s", dir, object)

	fileHandle, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("unable to open %s\nError: %s", filePath, err)
	}

	// Decode the file content
	gitObject := decodeGitObject(fileHandle)
	// The flag determines what information is returned
	switch flag {
	case "-p":
		return gitObject.Content, nil
	case "-t":
		return gitObject.Type, nil
	case "-s":
		return gitObject.Length, nil
	default:
		return "", fmt.Errorf("undefined flag for cat-file: %s", flag)
	}
}

func decodeGitObject(fileHandle *os.File) GitObject {
	// Put the file data in a buffer we can read from
	b := new(bytes.Buffer)
	_, err := io.Copy(b, fileHandle)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while reading from the file: %s", err)
		os.Exit(1)
	}
	// Decode the buffer data using a zlib reader
	r, err := zlib.NewReader(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while reading encoded data using zlib new reader: %s", err)
		os.Exit(1)
	}
	defer r.Close()
	// Read the data from the zlib reader
	decoded := make([]byte, 1024)
	count, _ := r.Read(decoded)
	decoded = decoded[:count]

	// Remove the null-byte after the length
	var n string
	for _, char := range string(decoded) {
		if char != '\x00' {
			n += string(char)
		}
	}
	regexContent := regexp.MustCompile(`^(\w+)\s(\d+)(.*)`)
	matches := regexContent.FindStringSubmatch(n)

	return GitObject{
		Type:    matches[1],
		Length:  matches[2],
		Content: matches[3],
	}
}
