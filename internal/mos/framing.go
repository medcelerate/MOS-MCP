package mos

import (
	"bufio"
	"encoding/xml"
	"io"

	"github.com/medcelerate/MOS-MCP/internal/mos/messages"
)

// StreamReader reads successive MOS <mos> documents from a byte stream.
//
// MOS peers keep a single socket open and send one XML document after another
// with no length prefix or delimiter beyond the XML itself. An xml.Decoder over
// a buffered reader consumes exactly one top-level element per Decode call and
// transparently handles reads that split a document across TCP segments, so it
// is a natural fit for this framing.
type StreamReader struct {
	dec *xml.Decoder
}

// NewStreamReader wraps r in a buffered MOS document reader.
func NewStreamReader(r io.Reader) *StreamReader {
	return &StreamReader{dec: xml.NewDecoder(bufio.NewReader(r))}
}

// Next decodes the next envelope from the stream. It returns io.EOF when the
// peer closes the connection cleanly between documents.
func (r *StreamReader) Next() (*messages.Envelope, error) {
	var e messages.Envelope
	if err := r.dec.Decode(&e); err != nil {
		return nil, err
	}
	return &e, nil
}
