package transcript

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
)

// maxLineSize bounds a single transcript line. Lines beyond this are
// discarded up to the next newline (observed Claude attachment lines exceed
// 30KB; 32MB leaves ample headroom while still bounding memory).
const maxLineSize = 32 * 1024 * 1024

// tailer reads complete lines appended to a file since the last drain.
// It never returns a torn line: a trailing fragment without a newline is
// carried over and re-joined on the next drain.
type tailer struct {
	path     string
	offset   int64
	carry    []byte
	skipping bool // discarding an oversized line until the next newline
}

func newTailer(path string) *tailer {
	return &tailer{path: path}
}

// Drain returns all complete lines appended since the previous call.
// A missing file is not an error (returns no lines).
func (t *tailer) Drain() ([][]byte, error) {
	f, err := os.Open(t.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	// Truncation/rotation guard: file shrank below our offset.
	if stat.Size() < t.offset {
		t.offset = 0
		t.carry = nil
		t.skipping = false
	}
	if _, err := f.Seek(t.offset, io.SeekStart); err != nil {
		return nil, err
	}

	reader := bufio.NewReader(f)
	var lines [][]byte
	for {
		chunk, err := reader.ReadBytes('\n')
		t.offset += int64(len(chunk))
		complete := len(chunk) > 0 && chunk[len(chunk)-1] == '\n'
		if complete {
			chunk = chunk[:len(chunk)-1]
		}

		if t.skipping {
			if complete {
				t.skipping = false
			}
		} else if len(t.carry)+len(chunk) > maxLineSize {
			t.carry = nil
			if !complete {
				t.skipping = true
			}
		} else if complete {
			line := chunk
			if len(t.carry) > 0 {
				line = append(t.carry, chunk...)
				t.carry = nil
			}
			line = bytes.TrimRight(line, "\r")
			if len(line) > 0 {
				lines = append(lines, line)
			}
		} else if len(chunk) > 0 {
			t.carry = append(t.carry, chunk...)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return lines, nil
			}
			return lines, err
		}
	}
}
