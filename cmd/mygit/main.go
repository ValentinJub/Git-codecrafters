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

	objects "github.com/codecrafters-io/git-starter-go/objects"
	utils "github.com/codecrafters-io/git-starter-go/utils"
)

type Dir string
type File string

// Usage: your_program.sh <command> <arg1> <arg2> ...
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
		// Initialize a new git repository, creating the necessary directories and files
		if err := utils.Mkdir(0755, ".git", ".git/objects", ".git/refs", ".git/hooks"); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
		}
		headFileContents := []byte("ref: refs/heads/main\n")
		configFileContents := []byte("[core]\nrepositoryformatversion = 0\nfilemode = true\nbare = false\nlogallrefupdates = true\nignorecase = true\nprecomposeunicode = true")
		err := utils.Mkfile([]string{".git/HEAD", ".git/config"}, [][]byte{headFileContents, configFileContents}, 0644)
		if err != nil {
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
		// Encode file to blob object (it's represents a file)
		res, err := hashObject()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while doing catfile stuff %s\n", err)
			os.Exit(1)
		}
		fmt.Print(res)
	case "ls-tree":
		// Decode a tree object and print its content
		res, err := listTree()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while processing ls-tree: %s\n", err)
			os.Exit(1)
		}
		fmt.Print(res)
	case "write-tree":
		// Write a tree object (it's represents a folder)
		tree, err := buildTree("./")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while building the tree: %s\n", err)
			os.Exit(1)
		}
		hash, err := writeObject(tree)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while writing the tree: %s\n", err)
			os.Exit(1)
		}
		fmt.Print(hash)
	case "commit-tree":
		// Write a commit object
		hash, err := writeCommit()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while commit tree: %s\n", err)
			os.Exit(1)
		}
		fmt.Print(hash)
	default:
		// Undefined command
		fmt.Fprintf(os.Stderr, "Undefined command %s\n", command)
		os.Exit(1)
	}
}

// $ git commit-tree 5b825dc642cb6eb9a060e54bf8d69288fbee4904 -p 3b18e512dba79e4c8300dd08aeb37f8e728b8dad -m "Second commit"
func writeCommit() (string, error) {
	if len(os.Args) < 7 {
		return "", fmt.Errorf("usage: mygit comit-tree <tree_sha> -p <parent_commit_sha> -m <commit_message>")
	}
	treeSha := os.Args[2]
	parentTreeSha := os.Args[4]
	message := os.Args[6]

	commit := objects.Commit{
		TreeSha:   []byte(treeSha),
		ParentSha: []byte(parentTreeSha),
		Message:   []byte(message),
	}
	content := commit.ToByteSlice()
	h, err := writeObject(content)
	if err != nil {
		return "", err
	}
	return h, nil
}

/*
Command: mygit write-tree

Writes the working directory in a tree object to the .git/objects directory
*/
func buildTree(root string) ([]byte, error) {
	// fmt.Printf("The root is %s\n", root)
	// The files & dirs in root
	dirs := make(map[string]bool)

	// Walk the root dir
	entries, err := os.ReadDir(root)
	if err != nil {
		return []byte{}, err
	}

	treeItems := make([]objects.TreeObjectItem, 0)
	// Map each dir/file found
	for _, entry := range entries {
		if entry.IsDir() {
			// fmt.Printf("%s/\n", entry.Name())
			if entry.Name() != ".git" {
				dirs[entry.Name()] = true
			} else {
				// fmt.Println("ignoring .git/ dir")
			}
		} else {
			// fmt.Printf("%s\n", entry.Name())
			// For each file, create a TreeObjectItem
			content, err := getBlobFromFile(root + entry.Name())
			if err != nil {
				return []byte{}, err
			}
			hash, err := calculateObjectHash(content)
			if err != nil {
				return []byte{}, err
			}
			treeItems = append(treeItems, objects.TreeObjectItem{
				Permission: "100644",
				Name:       entry.Name(),
				Sha1_Hash:  hash,
			})
		}
	}

	for dir := range dirs {
		tree, err := buildTree(dir + "/")
		if err != nil {
			return []byte{}, err
		}
		hash, err := calculateObjectHash(tree)
		if err != nil {
			return []byte{}, err
		}
		treeItems = append(treeItems, objects.TreeObjectItem{
			Permission: "40000",
			Name:       dir,
			Sha1_Hash:  hash,
		})
	}

	// Concat the file and dir in one content slice
	tree := objects.NewTreeObject(objects.ObjectHeader{}, treeItems...)
	return tree.ToByteSlice(), nil
}

// Encode the content and return its sha1 hash
func writeObject(content []byte) (string, error) {
	// Calculate the file hash
	hash, err := calculateObjectHash(content)
	if err != nil {
		return "", err
	}
	sha1_hash := hex.EncodeToString(hash)

	dir := fmt.Sprintf(".git/objects/%s", sha1_hash[:2])
	fileName := sha1_hash[2:]
	fullPath := fmt.Sprintf("%s/%s", dir, fileName)

	// Print the encoded data in the new file
	b := new(bytes.Buffer)
	zlibWriter := zlib.NewWriter(b)
	_, err = zlibWriter.Write(content)
	if err != nil {
		return "", fmt.Errorf("error while writing using the zlib writer: %s", err)
	}
	zlibWriter.Close()
	compressed := b.Bytes()

	// Create the dir and the file with the encoded data
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("error creating directory: %s", err)
	}
	if err := os.WriteFile(fullPath, compressed, 0644); err != nil {
		return "", fmt.Errorf("error while writing file: %s", err)
	}

	return sha1_hash, nil
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
		sha1_hash, err := encodeBlobObject(file)
		if err != nil {
			return "", fmt.Errorf("error while encodng objects.BlobObject: %s", err)
		}
		return sha1_hash, nil
	default:
		return "", fmt.Errorf("error: unknown flag passed: %s, authorised: [w]", flag)
	}
}

// Display content, size or type of a git/objects
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
	blobObj, err := decodeBlobObject(filePath)
	if err != nil {
		return "", fmt.Errorf("error while decoding blobObj: %s", err)
	}
	// The flag determines what information is returned
	switch flag {
	case "-p":
		return blobObj.Content, nil
	case "-t":
		return blobObj.Type, nil
	case "-s":
		return blobObj.Length, nil
	default:
		return "", fmt.Errorf("undefined flag for cat-file: %s", flag)
	}
}

// Create a blob object and returns the sha1_sum and an error if there was any
func encodeBlobObject(file string) (string, error) {
	blobSlice, err := getBlobFromFile(file)
	if err != nil {
		return "", err
	}
	h, err := writeObject(blobSlice)
	if err != nil {
		return "", err
	}
	return h, nil
}

// Return the content of a file formatted in a blob fashion: <type> <size>\x00<content>
func getBlobFromFile(file string) ([]byte, error) {
	b, err := utils.ReadFile(file)
	if err != nil {
		return []byte{}, err
	}
	blob := objects.NewBlobObject(
		objects.ObjectHeader{
			Type:   "blob",
			Length: fmt.Sprintf("%d", b.Len()),
		},
		b.String(),
	)
	return blob.ToByteSlice(), nil
}

// Return the sha1hash of the content
func calculateObjectHash(content []byte) ([]byte, error) {
	h := sha1.New()
	if _, e := h.Write(content); e != nil {
		return []byte{}, e
	}
	return h.Sum(nil), nil
}

// Return human readable values from a zlib encoded blob object
func decodeBlobObject(filePath string) (objects.BlobObject, error) {
	fileHandle, err := os.Open(filePath)
	if err != nil {
		return objects.BlobObject{}, fmt.Errorf("unable to open %s\nError: %s", filePath, err)
	}
	decoded, err := decodeFileWithZlib(fileHandle)
	if err != nil {
		return objects.BlobObject{}, fmt.Errorf("unable to decode, error: %s", err)
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

	return objects.BlobObject{
		ObjectHeader: objects.ObjectHeader{
			Type:   matches[1],
			Length: matches[2],
		},
		Content: matches[3],
	}, nil
}

// Decode a zlib encoded content from file
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
