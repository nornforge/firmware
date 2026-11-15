// Package receiver reads norn commands from the serial line and keeps the
// firmware in sync with the host when a frame is cut off.
//
// A frame from the host arrives in one go: about 70 bytes, a few hundred
// microseconds over USB CDC. A pause in the middle of a frame therefore means
// that the host stopped sending (a CLI killed mid-write, a cable unplugged, a
// USB reset). Without a timeout the parser would wait for the missing bytes
// and take them from the next command, which then fails although the host sent
// it correctly. The Receiver discards the bytes of such a frame instead.
package receiver

import (
	"bufio"
	"errors"
	"time"

	"github.com/nornforge/norn/pkg/norn"
)

// DefaultTimeout is the longest pause between two bytes of the same frame.
const DefaultTimeout = 100 * time.Millisecond

// ErrCutOff is returned by Next if a frame stayed incomplete for longer than
// the timeout. Its bytes have been discarded; the host is no longer waiting
// for an answer, so none should be sent.
var ErrCutOff = errors.New("frame cut off, received bytes discarded")

// Source is the serial line. machine.Serial satisfies it.
type Source interface {
	// Buffered returns the number of bytes that can be read without waiting.
	Buffered() int
	// ReadByte returns the next byte; it is only called if Buffered is > 0.
	ReadByte() (byte, error)
}

// Receiver reads commands from a Source.
type Receiver struct {
	src     Source
	timeout time.Duration
	now     func() time.Time
	wait    func()
	reader  *bufio.Reader

	// last is the time the most recent byte was received.
	last time.Time
	// inFrame is set while bytes of a frame have been received but the frame
	// is not parsed yet.
	inFrame bool
	// cutOff is set by Read when the timeout has expired.
	cutOff bool
}

// New returns a Receiver reading from src. now returns the current time and
// wait is called while no byte is available, to give the CPU to other
// goroutines.
func New(src Source, timeout time.Duration, now func() time.Time, wait func()) *Receiver {
	r := &Receiver{src: src, timeout: timeout, now: now, wait: wait}
	r.reader = bufio.NewReader((*byteReader)(r))
	return r
}

// Next blocks until the next command is received. It returns ErrCutOff if a
// frame was discarded because the host stopped sending in the middle of it,
// and the parse error for any other invalid input.
func (r *Receiver) Next(command *norn.Command) error {
	// Bytes still in the buffer belong to the next frame.
	r.inFrame = r.reader.Buffered() > 0
	r.cutOff = false
	err := command.Parse(r.reader)
	if r.cutOff {
		r.reader.Reset((*byteReader)(r))
		r.inFrame = false
		return ErrCutOff
	}
	return err
}

// byteReader is the io.Reader below the bufio.Reader. It is a separate type so
// that Read is not part of the Receiver's API.
type byteReader Receiver

// Read blocks until at least one byte is available. While a frame is
// incomplete, it gives up once no byte has arrived for the timeout.
func (b *byteReader) Read(p []byte) (int, error) {
	r := (*Receiver)(b)
	if len(p) == 0 {
		return 0, nil
	}
	for {
		if n := r.src.Buffered(); n > 0 {
			if n > len(p) {
				n = len(p)
			}
			for i := 0; i < n; i++ {
				c, err := r.src.ReadByte()
				if err != nil {
					return i, err
				}
				p[i] = c
			}
			r.last = r.now()
			r.inFrame = true
			return n, nil
		}
		if r.inFrame && r.now().Sub(r.last) >= r.timeout {
			r.cutOff = true
			return 0, ErrCutOff
		}
		r.wait()
	}
}
