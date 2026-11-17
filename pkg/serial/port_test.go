package serial_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bugstserial "go.bug.st/serial"

	"github.com/MyChaOS87/reverseLCN/pkg/serial"
	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker/packet"
)

var (
	errFakeOpen    = errors.New("fake open error")
	errFakeConfig  = errors.New("fake set timeout error")
	errFakeRead    = errors.New("fake read error")
	errFakeClosed  = errors.New("fake port closed")
	errFakeInvalid = packet.ErrPacketInvalid
	errFakeShort   = packet.ErrPacketIncomplete
)

type readStep struct {
	data []byte
	err  error
}

type fakeSerialDevice struct {
	mu         sync.Mutex
	reads      chan readStep
	written    [][]byte
	writtenCh  chan []byte
	timeout    time.Duration
	setTimeErr error
	closed     bool
	closeOnce  sync.Once
	closedCh   chan struct{}
}

func newFakeSerialDevice(steps ...readStep) *fakeSerialDevice {
	ch := make(chan readStep, len(steps)+1)
	for _, s := range steps {
		ch <- s
	}

	return &fakeSerialDevice{
		reads:     ch,
		writtenCh: make(chan []byte, 128),
		closedCh:  make(chan struct{}),
	}
}

func (f *fakeSerialDevice) SetMode(_ *bugstserial.Mode) error { return nil }
func (f *fakeSerialDevice) Drain() error                      { return nil }
func (f *fakeSerialDevice) ResetInputBuffer() error           { return nil }
func (f *fakeSerialDevice) ResetOutputBuffer() error          { return nil }
func (f *fakeSerialDevice) SetDTR(_ bool) error               { return nil }
func (f *fakeSerialDevice) SetRTS(_ bool) error               { return nil }
func (f *fakeSerialDevice) GetModemStatusBits() (*bugstserial.ModemStatusBits, error) {
	return &bugstserial.ModemStatusBits{}, nil
}
func (f *fakeSerialDevice) Break(_ time.Duration) error { return nil }

func (f *fakeSerialDevice) SetReadTimeout(t time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.timeout = t

	return f.setTimeErr
}

func (f *fakeSerialDevice) Read(p []byte) (int, error) {
	select {
	case step := <-f.reads:
		if step.err != nil {
			return 0, step.err
		}

		n := copy(p, step.data)

		return n, nil
	case <-f.closedCh:
		return 0, errFakeClosed
	}
}

func (f *fakeSerialDevice) Write(p []byte) (int, error) {
	cp := slices.Clone(p)

	f.mu.Lock()
	f.written = append(f.written, cp)
	f.mu.Unlock()

	select {
	case f.writtenCh <- cp:
	default:
	}

	return len(p), nil
}

func (f *fakeSerialDevice) Close() error {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()

	f.closeOnce.Do(func() {
		close(f.closedCh)
	})

	return nil
}

func (f *fakeSerialDevice) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.closed
}

func (f *fakeSerialDevice) writtenFrames() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.written)
}

type fakePacket []byte

func (p *fakePacket) Serialize() ([]byte, error) { return slices.Clone(*p), nil }
func (p *fakePacket) ToString() string           { return string(*p) }
func (p *fakePacket) ToNiceString() string       { return string(*p) }

// twoByteHeaderDeserialize parses [tag, expectedLen] frames where expectedLen is 2 or 6.
func twoByteHeaderDeserialize(buf []byte) (packet.Packet, int, error) {
	const (
		minLen  = 2
		longLen = 6
	)

	if len(buf) < minLen {
		return nil, 0, errFakeShort
	}

	expectedLen := int(buf[1])
	if expectedLen != minLen && expectedLen != longLen {
		return nil, 0, errFakeInvalid
	}

	if len(buf) < expectedLen {
		return nil, 0, errFakeShort
	}

	p := fakePacket(slices.Clone(buf[:expectedLen]))

	return &p, expectedLen, nil
}

func TestPortSendFIFOAndQueueBounds(t *testing.T) {
	t.Parallel()

	t.Run("preserves strict FIFO order and isolates caller slice mutations", func(t *testing.T) {
		t.Parallel()

		const count = 50

		dev := newFakeSerialDevice()
		p := serial.NewPort(
			serial.PortName("/dev/fake0"),
			serial.ReadTimeout(5*time.Millisecond),
			serial.PortOpener(func(_ string, _ *bugstserial.Mode) (bugstserial.Port, error) {
				return dev, nil
			}),
		)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		runErr := make(chan error, 1)

		go func() {
			runErr <- p.Run(ctx, func(_ packet.Packet) {})
		}()

		want := make([][]byte, 0, count)
		scratch := []byte{0x01, 0x00}

		for i := range count {
			scratch[1] = byte(i)
			want = append(want, []byte{0x01, byte(i)})

			require.NoError(t, p.Send(ctx, scratch))
			// Mutate scratch immediately after Send to confirm defensive cloning.
			scratch[1] = 0xFF
		}

		for range count {
			<-dev.writtenCh
		}

		cancel()
		require.NoError(t, <-runErr)
		assert.True(t, dev.isClosed())
		assert.Equal(t, want, dev.writtenFrames())
	})

	t.Run("returns ErrSendQueueFull when outbound queue reaches capacity", func(t *testing.T) {
		t.Parallel()

		p := serial.NewPort(serial.SendQueueSize(2))
		ctx := t.Context()

		require.NoError(t, p.Send(ctx, []byte{1}))
		require.NoError(t, p.Send(ctx, []byte{2}))

		err := p.Send(ctx, []byte{3})
		require.ErrorIs(t, err, serial.ErrSendQueueFull)
	})

	t.Run("returns context error when Send called with cancelled context", func(t *testing.T) {
		t.Parallel()

		p := serial.NewPort(serial.SendQueueSize(2))
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := p.Send(ctx, []byte{1, 2})
		require.ErrorIs(t, err, context.Canceled)
	})
}

func TestPortRunLifecycleAndIdleFlush(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupDev  func() (*fakeSerialDevice, serial.PortOpenerFunc)
		wantErr   error
		wantEject []packet.Packet
	}{
		{
			name: "open error returns ErrPortOpen",
			setupDev: func() (*fakeSerialDevice, serial.PortOpenerFunc) {
				return nil, func(_ string, _ *bugstserial.Mode) (bugstserial.Port, error) {
					return nil, errFakeOpen
				}
			},
			wantErr: serial.ErrPortOpen,
		},
		{
			name: "SetReadTimeout error closes device and returns ErrPortConfig",
			setupDev: func() (*fakeSerialDevice, serial.PortOpenerFunc) {
				dev := newFakeSerialDevice()
				dev.setTimeErr = errFakeConfig

				return dev, func(_ string, _ *bugstserial.Mode) (bugstserial.Port, error) {
					return dev, nil
				}
			},
			wantErr: serial.ErrPortConfig,
		},
		{
			name: "idle read timeout flushes stalled frame behind noise byte before read error",
			setupDev: func() (*fakeSerialDevice, serial.PortOpenerFunc) {
				dev := newFakeSerialDevice(
					readStep{data: []byte{0xFF, 0x06, 0x02}},
					readStep{data: nil}, // 100ms timeout (n == 0, err == nil)
					readStep{err: errFakeRead},
				)

				return dev, func(_ string, _ *bugstserial.Mode) (bugstserial.Port, error) {
					return dev, nil
				}
			},
			wantErr: serial.ErrPortRead,
			wantEject: []packet.Packet{
				&fakePacket{0x06, 0x02},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dev, opener := tt.setupDev()
			p := serial.NewPort(
				serial.PortName("/dev/fake1"),
				serial.Deserializer(twoByteHeaderDeserialize),
				serial.MinLength(2),
				serial.PortOpener(opener),
			)

			var got []packet.Packet

			err := p.Run(t.Context(), func(pkt packet.Packet) {
				got = append(got, pkt)
			})
			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantEject, got)

			if dev != nil {
				assert.True(t, dev.isClosed())
			}
		})
	}
}
