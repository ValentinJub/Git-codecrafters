package objects

import "fmt"

type ObjectHeader struct {
	Type   string
	Length string
}

func (h *ObjectHeader) ToByteSlice() []byte {
	return []byte(fmt.Sprintf("%s %s\x00", h.Type, h.Length))
}
