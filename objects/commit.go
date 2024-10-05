package objects

import (
	"fmt"
)

const (
	AUTHOR       = "Valentin"
	AUTHOR_EMAIL = "valentinwissler42@outlook.com"
	A_DATE_SEC   = "946684800"
	A_TIMEZONE   = "+0000"
)

// https://stackoverflow.com/questions/22968856/what-is-the-file-format-of-a-git-commit-object-data-structure
type Commit struct {
	ObjectHeader
	TreeSha   []byte
	ParentSha []byte
	Message   []byte
}

func (c *Commit) ToByteSlice() []byte {
	content := make([]byte, 0)
	content = append(content, []byte(fmt.Sprintf("tree %s\n", string(c.TreeSha)))...)
	content = append(content, []byte(fmt.Sprintf("parent %s\n", string(c.ParentSha)))...)
	content = append(content, []byte(fmt.Sprintf("author %s <%s> %s %s\n", AUTHOR, AUTHOR_EMAIL, A_DATE_SEC, A_TIMEZONE))...)
	content = append(content, []byte(fmt.Sprintf("commiter %s <%s> %s %s\n\n", AUTHOR, AUTHOR_EMAIL, A_DATE_SEC, A_TIMEZONE))...)
	content = append(content, []byte(fmt.Sprintf("%s\n", c.Message))...)
	header := ObjectHeader{
		Type:   "commit",
		Length: fmt.Sprintf("%d", len(content)),
	}
	return append(header.ToByteSlice(), content...)
}
