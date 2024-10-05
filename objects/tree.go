package objects

import (
	"fmt"
	"sort"
)

type TreeObject struct {
	ObjectHeader
	Items []TreeObjectItem
}

type TreeObjectItem struct {
	Permission, Name string
	Sha1_Hash        []byte
}

func NewTreeObject(header ObjectHeader, items ...TreeObjectItem) TreeObject {
	return TreeObject{
		ObjectHeader: header,
		Items:        items,
	}
}

func (t *TreeObject) ToByteSlice() []byte {
	sort.Slice(t.Items, func(i, j int) bool {
		return t.Items[i].Name < t.Items[j].Name
	})
	content := make([]byte, 0)
	for _, item := range t.Items {
		content = append(content, item.ToByteSlice()...)
	}
	header := ObjectHeader{
		Type:   "tree",
		Length: fmt.Sprintf("%d", len(content)),
	}
	t.ObjectHeader = header
	return append(header.ToByteSlice(), content...)
}

func NewTreeObjectItem(name string, hash []byte) (TreeObjectItem, error) {
	if len(hash) != 20 {
		return TreeObjectItem{}, fmt.Errorf("unexpected hash length: %d, expected: 20", len(hash))
	}
	switch name {
	case "tree":
		return TreeObjectItem{
			Name:       name,
			Permission: "100644",
			Sha1_Hash:  hash,
		}, nil
	case "blob":
		return TreeObjectItem{
			Name:       name,
			Permission: "040000",
			Sha1_Hash:  hash,
		}, nil
	default:
		return TreeObjectItem{}, fmt.Errorf("unexpected type: %s", name)
	}
}

// Return the byte slice representation of this tree item
// <mode> <name>\x00<sha1_hash>
func (tr *TreeObjectItem) ToByteSlice() []byte {
	return append([]byte(fmt.Sprintf("%s %s\x00", tr.Permission, tr.Name)), tr.Sha1_Hash[:]...)
}
