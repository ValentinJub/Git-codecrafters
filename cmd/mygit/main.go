package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type GitObjectHeader struct {
	Type   string
	Length string
}

func (h *GitObjectHeader) ToByteSlice() []byte {
	return []byte(fmt.Sprintf("%s %s\x00", h.Type, h.Length))
}

type GitObjectBlob struct {
	GitObjectHeader
	Content string
}

type GitCommand struct {
	Args []string
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
	case "hash-object":
		// Display information about .git/objects
		res, err := hashObject()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while doing catfile stuff %s\n", err)
			os.Exit(1)
		}
		fmt.Print(res)
	case "ls-tree":
		res, err := listTree()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while processing ls-tree: %s\n", err)
			os.Exit(1)
		}
		fmt.Print(res)
	default:
		// Undefined command
		fmt.Fprintf(os.Stderr, "Undefined command %s\n", command)
		os.Exit(1)
	}
}

// git ls-tree --name-only <tree_sha>
//
//	tree <size>\0
//
// <mode> <name>\0<20_byte_sha>
// <mode> <name>\0<20_byte_sha>
func listTree() (string, error) {
	if len(os.Args) < 4 {
		return "", fmt.Errorf("usage: mygit ls-tree <flags> <Treeobjects>")
	}
	fileHandle, err := os.Open(".git/objects/" + os.Args[3][:2] + "/" + os.Args[3][2:])
	if err != nil {
		return "", fmt.Errorf("unable to open %s\nError: %s", os.Args[3], err)
	}
	content, err := decodeFileWithZlib(fileHandle)
	if err != nil {
		return "", fmt.Errorf("error while decoding file with zlib: %s", err)
	}
	switch flag := os.Args[2]; flag {
	case "--name-only":
		treeContentNames, err := decodeTreeContentNames(content)
		if err != nil {
			return "", fmt.Errorf("error while decoding tree content: %s", err)
		}
		// fmt.Printf("The content of the file to string is:\n%s", string(content))
		return strings.Join(treeContentNames, "\n") + "\n", nil
	default:
		return "", fmt.Errorf("unknown flag passed: %s", flag)
	}
}

// return the names contained in the Tree
func decodeTreeContentNames(content []byte) ([]string, error) {
	// all the tree/blob names are before the null (\x00) byte, minus the first one
	// we can collect all index of the null bytes, and grab all the chars before it until it finds a whitespace
	indices := make([]int, 0)
	for i, c := range content {
		if c == '\x00' {
			indices = append(indices, i)
		}
	}
	indices = indices[1:]
	names := make([]string, 0)
	for _, i := range indices {
		name := make([]byte, 0)
		i--
		for {
			if content[i] == ' ' {
				break
			}
			name = append([]byte{content[i]}, name...)
			i--
		}
		names = append(names, string(name))
	}
	return names, nil
}

// hash-object -w test.txt
func hashObject() (string, error) {
	if len(os.Args) < 4 {
		return "", fmt.Errorf("usage: mygit hash-object <flags> <objects>")
	}
	switch flag := os.Args[2]; flag {
	case "-w":
		file := os.Args[3]
		sha1_hash, err := encodeGitObjectBlob(file)
		if err != nil {
			return "", fmt.Errorf("error while encodng GitObjectBlob: %s", err)
		}
		return sha1_hash, nil
	default:
		return "", fmt.Errorf("error: unknown flag passed: %s, authorised: [w]", flag)
	}
}

// Display content, size or type of a git/object
// The content is encoded using zlib
// Example: mygit cat-file -p 4csejhtq23098ughaohjg
func catFile(args []string) (string, error) {
	if len(args) < 4 {
		return "", fmt.Errorf("usage: mygit cat-file <flags> <objects>")
	}

	flag := args[2]
	file := args[3]
	dir := string(file[:2])
	object := string(file[2:])
	filePath := fmt.Sprintf(".git/objects/%s/%s", dir, object)

	// Decode the file content
	GitObjectBlob, err := decodeGitObjectBlob(filePath)
	if err != nil {
		return "", fmt.Errorf("error while decoding GitObjectBlob: %s", err)
	}
	// The flag determines what information is returned
	switch flag {
	case "-p":
		return GitObjectBlob.Content, nil
	case "-t":
		return GitObjectBlob.Type, nil
	case "-s":
		return GitObjectBlob.Length, nil
	default:
		return "", fmt.Errorf("undefined flag for cat-file: %s", flag)
	}
}

// Returns the sha1_sum and an error if there was any
func encodeGitObjectBlob(file string) (string, error) {
	fileHandle, err := os.Open(file)
	if err != nil {
		return "", fmt.Errorf("unable to open %s\nError: %s", file, err)
	}
	// Put the file data in a buffer we can read from
	b := new(bytes.Buffer)
	_, err = io.Copy(b, fileHandle)
	if err != nil {
		return "", fmt.Errorf("error while reading from the file: %s", err)
	}
	// Create the content header
	header := []byte(fmt.Sprintf("blob %d\x00", b.Len()))
	// Merge the header with the content
	content := append(header, b.Bytes()...)

	// Compute the sha hash of the file
	h := sha1.New()
	h.Write(content)
	sha1_hash := hex.EncodeToString(h.Sum(nil))

	dir := fmt.Sprintf(".git/objects/%s", sha1_hash[:2])
	fileName := sha1_hash[2:]
	fullPath := fmt.Sprintf("%s/%s", dir, fileName)

	b = new(bytes.Buffer)
	zlibWriter := zlib.NewWriter(b)
	_, err = zlibWriter.Write([]byte(content))
	if err != nil {
		return "", fmt.Errorf("error while writing using the zlib writer: %s", err)
	}
	zlibWriter.Close()
	compressed := b.Bytes()

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("error creating directory: %s", err)
	}
	if err := os.WriteFile(fullPath, compressed, 0644); err != nil {
		return "", fmt.Errorf("error while writing file: %s", err)
	}

	return sha1_hash, nil
}

func decodeGitObjectBlob(filePath string) (GitObjectBlob, error) {
	fileHandle, err := os.Open(filePath)
	if err != nil {
		return GitObjectBlob{}, fmt.Errorf("unable to open %s\nError: %s", filePath, err)
	}
	decoded, err := decodeFileWithZlib(fileHandle)
	if err != nil {
		return GitObjectBlob{}, fmt.Errorf("unable to decode, error: %s", err)
	}
	// Remove the null-byte after the length
	var n string
	for _, char := range string(decoded) {
		if char != '\x00' {
			n += string(char)
		}
	}

	regexContent := regexp.MustCompile(`^(\w+)\s(\d+)(.*)`)
	matches := regexContent.FindStringSubmatch(n)

	return GitObjectBlob{
		GitObjectHeader: GitObjectHeader{
			Type:   matches[1],
			Length: matches[2],
		},
		Content: matches[3],
	}, nil
}

func decodeFileWithZlib(file *os.File) ([]byte, error) {
	// Put the file data in a buffer we can read from
	b := new(bytes.Buffer)
	_, err := io.Copy(b, file)
	if err != nil {
		return []byte{}, fmt.Errorf("error while reading from the file: %s", err)
	}
	// Decode the buffer data using a zlib reader
	r, err := zlib.NewReader(b)
	if err != nil {
		return []byte{}, fmt.Errorf("error while reading encoded data using zlib new reader: %s", err)
	}
	defer r.Close()
	// Read the data from the zlib reader
	decoded := make([]byte, 1024)
	count, _ := r.Read(decoded)
	decoded = decoded[:count]
	return decoded, nil
}
