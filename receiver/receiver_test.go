package receiver_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nornforge/firmware/receiver"
	"github.com/nornforge/norn/pkg/norn"
)

// chunk is a piece of input that arrives at the given time.
type chunk struct {
	at   time.Duration
	data string
}

// line simulates the serial line with a fake clock. The clock only moves
// forward when the receiver waits.
type line struct {
	now     time.Duration
	pending []chunk
	rx      []byte
}

func (l *line) deliver() {
	for len(l.pending) > 0 && l.pending[0].at <= l.now {
		l.rx = append(l.rx, l.pending[0].data...)
		l.pending = l.pending[1:]
	}
}

func (l *line) Buffered() int {
	l.deliver()
	return len(l.rx)
}

func (l *line) ReadByte() (byte, error) {
	if len(l.rx) == 0 {
		return 0, errors.New("buffer empty")
	}
	c := l.rx[0]
	l.rx = l.rx[1:]
	return c, nil
}

func (l *line) clock() time.Time { return time.Unix(0, 0).Add(l.now) }

func (l *line) wait() {
	if len(l.pending) == 0 {
		panic("receiver waits for input that never arrives")
	}
	l.now += time.Millisecond
}

func newReceiver(chunks ...chunk) (*receiver.Receiver, *line) {
	l := &line{pending: chunks}
	return receiver.New(l, receiver.DefaultTimeout, l.clock, l.wait), l
}

func frame(t norn.CommandType, channel uint) string {
	c := norn.Command{Type: t, Channel: channel}
	return string(c.Marshal())
}

func expectCommand(t *testing.T, r *receiver.Receiver, want norn.CommandType, channel uint) {
	t.Helper()
	var c norn.Command
	if err := r.Next(&c); err != nil {
		t.Fatalf("Next() = %v, want command %v", err, want)
	}
	if c.Type != want || c.Channel != channel {
		t.Fatalf("Next() = %+v, want type %v channel %d", c, want, channel)
	}
}

func expectCutOff(t *testing.T, r *receiver.Receiver) {
	t.Helper()
	var c norn.Command
	if err := r.Next(&c); !errors.Is(err, receiver.ErrCutOff) {
		t.Fatalf("Next() = %v, want %v", err, receiver.ErrCutOff)
	}
}

func TestCompleteFrame(t *testing.T) {
	r, _ := newReceiver(chunk{0, frame(norn.Set, 3)})
	expectCommand(t, r, norn.Set, 3)
}

func TestFrameAfterLongIdle(t *testing.T) {
	// Waiting for the first byte of a frame never times out.
	r, _ := newReceiver(chunk{10 * time.Second, frame(norn.Get, 1)})
	expectCommand(t, r, norn.Get, 1)
}

func TestFrameWithShortPause(t *testing.T) {
	f := frame(norn.Set, 2)
	r, _ := newReceiver(chunk{0, f[:20]}, chunk{receiver.DefaultTimeout / 2, f[20:]})
	expectCommand(t, r, norn.Set, 2)
}

func TestCutOffFrameIsDiscarded(t *testing.T) {
	f := frame(norn.Set, 2)
	r, l := newReceiver(chunk{0, f[:30]}, chunk{time.Second, frame(norn.Version, 0)})
	expectCutOff(t, r)
	if l.now < receiver.DefaultTimeout || l.now >= time.Second {
		t.Fatalf("cut off detected at %v, want between %v and 1s", l.now, receiver.DefaultTimeout)
	}
	// The next command is answered on its own, not mixed with the cut-off bytes.
	expectCommand(t, r, norn.Version, 0)
}

func TestStrayBytesAreDiscarded(t *testing.T) {
	// ModemManager sends this kind of probe to new ttyACM devices.
	r, _ := newReceiver(chunk{0, "AT\r"}, chunk{time.Second, frame(norn.Version, 0)})
	expectCutOff(t, r)
	expectCommand(t, r, norn.Version, 0)
}

func TestTwoFramesInOneChunk(t *testing.T) {
	// The second frame is already in the buffer when the first one is parsed;
	// it must be parsed without waiting for more input.
	r, _ := newReceiver(chunk{0, frame(norn.Set, 1) + frame(norn.Get, 1)})
	expectCommand(t, r, norn.Set, 1)
	expectCommand(t, r, norn.Get, 1)
}

func TestFrameCutOffAfterCompleteFrame(t *testing.T) {
	// A complete frame followed by the start of another one in the same chunk.
	f := frame(norn.Set, 4)
	r, _ := newReceiver(
		chunk{0, frame(norn.Set, 1) + f[:25]},
		chunk{time.Second, frame(norn.Get, 4)},
	)
	expectCommand(t, r, norn.Set, 1)
	expectCutOff(t, r)
	expectCommand(t, r, norn.Get, 4)
}

func TestInvalidFrameIsAParseError(t *testing.T) {
	f := []byte(frame(norn.Set, 1))
	f[len(f)-1] ^= 1 // break the payload checksum
	r, _ := newReceiver(chunk{0, string(f)})
	var c norn.Command
	err := r.Next(&c)
	if err == nil || errors.Is(err, receiver.ErrCutOff) {
		t.Fatalf("Next() = %v, want a parse error", err)
	}
}
