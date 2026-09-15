package mos

import (
	"io"
	"testing"
)

// slowReader releases the byte stream in small fixed-size chunks to exercise
// the decoder's handling of reads that split MOS documents.
type slowReader struct {
	data  []byte
	chunk int
	pos   int
}

func (r *slowReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	end := r.pos + r.chunk
	if end > len(r.data) {
		end = len(r.data)
	}
	n := copy(p, r.data[r.pos:end])
	r.pos += n
	return n, nil
}

const twoMessages = `<mos><mosID>a</mosID><ncsID>b</ncsID><messageID>1</messageID><heartbeat><time>t1</time></heartbeat></mos>` +
	`<mos><mosID>a</mosID><ncsID>b</ncsID><messageID>2</messageID><roDelete><roID>RO9</roID></roDelete></mos>`

func TestStreamReaderConcatenated(t *testing.T) {
	for _, chunk := range []int{1, 7, 64, 4096} {
		r := NewStreamReader(&slowReader{data: []byte(twoMessages), chunk: chunk})

		first, err := r.Next()
		if err != nil {
			t.Fatalf("chunk=%d first: %v", chunk, err)
		}
		if first.BodyName() != "heartbeat" || first.MessageID != "1" {
			t.Fatalf("chunk=%d first = %s/%s", chunk, first.BodyName(), first.MessageID)
		}

		second, err := r.Next()
		if err != nil {
			t.Fatalf("chunk=%d second: %v", chunk, err)
		}
		if second.BodyName() != "roDelete" || second.RoDelete.RoID != "RO9" {
			t.Fatalf("chunk=%d second = %s roID=%q", chunk, second.BodyName(), second.RoDelete.RoID)
		}

		if _, err := r.Next(); err != io.EOF {
			t.Fatalf("chunk=%d expected EOF, got %v", chunk, err)
		}
	}
}
