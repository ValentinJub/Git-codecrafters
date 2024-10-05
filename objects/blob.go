package objects

type BlobObject struct {
	ObjectHeader
	Content string
}

func NewBlobObject(header ObjectHeader, content string) BlobObject {
	return BlobObject{
		ObjectHeader: header,
		Content:      content,
	}
}

func (b *BlobObject) ToByteSlice() []byte {
	return append(b.ObjectHeader.ToByteSlice(), b.Content...)
}
