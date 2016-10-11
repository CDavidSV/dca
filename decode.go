package dca

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

var (
	ErrNotDCA        = errors.New("DCA Magic header not found, either not dca or raw dca frames")
	ErrNotFirstFrame = errors.New("Metadata can only be found in the first frame")
)

type Decoder struct {
	Metadata      *Metadata
	FormatVersion int
	r             *bufio.Reader

	// Set to true after the first frame has been read
	firstFrameProcessed bool
}

// NewDecoder returns a new dca decoder
func NewDecoder(r io.Reader) *Decoder {
	decoder := &Decoder{
		r: bufio.NewReader(r),
	}

	return decoder
}

// ReadMetadata reads the first metadata frame
// OpusFrame will call this automatically if
func (d *Decoder) ReadMetadata() error {
	if d.firstFrameProcessed {
		return ErrNotFirstFrame
	}
	d.firstFrameProcessed = true

	fingerprint := make([]byte, 4)
	_, err := d.r.Read(fingerprint)
	if err != nil {
		return err
	}

	if string(fingerprint[:3]) != "DCA" {
		return ErrNotDCA
	}

	// Read the format version
	version, err := strconv.ParseInt(string(fingerprint[3:]), 10, 32)
	if err != nil {
		return err
	}
	d.FormatVersion = int(version)

	// The length of the metadata
	var metaLen int32
	err = binary.Read(d.r, binary.LittleEndian, &metaLen)
	if err != nil {
		return err
	}

	// Read in the metadata itself
	jsonBuf := make([]byte, metaLen)
	err = binary.Read(d.r, binary.LittleEndian, &jsonBuf)
	if err != nil {
		return err
	}

	// And unmarshal it
	var metadata *Metadata
	err = json.Unmarshal(jsonBuf, &metadata)
	d.Metadata = metadata
	return err
}

// OpusFrame returns the next audio frame
// If this is the first frame it will also check for metadata in it
func (d *Decoder) OpusFrame() (frame []byte, err error) {
	if !d.firstFrameProcessed {
		// Check to see if this contains metadata and read the metadata if so
		magic, err := d.r.Peek(3)
		if err != nil {
			return nil, err
		}

		if string(magic) == "DCA" {
			err = d.ReadMetadata()
			if err != nil {
				return nil, err
			}
		}
	}

	frame, err = DecodeFrame(d.r)
	return
}