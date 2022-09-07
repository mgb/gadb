package gadb

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strconv"
	"time"
)

// ErrConnBroken is returned when the connection is broken
var ErrConnBroken = errors.New("socket connection broken")

const (
	defaultAdbReadTimeout = 60 * time.Second
)

type transport struct {
	sock        net.Conn
	readTimeout time.Duration
}

func newTransport(address string) (transport, error) {
	tp := transport{
		readTimeout: defaultAdbReadTimeout,
	}

	var err error
	tp.sock, err = net.Dial("tcp", address)
	if err != nil {
		return tp, fmt.Errorf("adb transport: %w", err)
	}
	return tp, nil
}

func (t transport) Send(command string) error {
	msg := fmt.Sprintf("%04x%s", len(command), command)
	return _send(t.sock, []byte(msg))
}

func (t transport) VerifyResponse() error {
	status, err := t.ReadStringN(4)
	if err != nil {
		return err
	}
	if status == "OKAY" {
		return nil
	}

	sError, err := t.UnpackString()
	if err != nil {
		return err
	}
	return fmt.Errorf("command failed: %s", sError)
}

func (t transport) ReadStringAll() (string, error) {
	raw, err := t.ReadBytesAll()
	return string(raw), err
}

func (t transport) ReadBytesAll() ([]byte, error) {
	return ioutil.ReadAll(t.sock)
}

func (t transport) UnpackString() (string, error) {
	raw, err := t.UnpackBytes()
	return string(raw), err
}

func (t transport) UnpackBytes() ([]byte, error) {
	length, err := t.ReadStringN(4)
	if err != nil {
		return nil, err
	}

	size, err := strconv.ParseInt(length, 16, 64)
	if err != nil {
		return nil, err
	}

	return t.ReadBytesN(int(size))
}

func (t transport) ReadStringN(size int) (string, error) {
	raw, err := t.ReadBytesN(size)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (t transport) ReadBytesN(size int) ([]byte, error) {
	_ = t.sock.SetReadDeadline(time.Now().Add(t.readTimeout))
	return _readN(t.sock, size)
}

func (t transport) Close() error {
	if t.sock == nil {
		return nil
	}
	return t.sock.Close()
}

func (t transport) CreateSyncTransport() (syncTransport, error) {
	err := t.Send("sync:")
	if err != nil {
		return syncTransport{}, err
	}

	err = t.VerifyResponse()
	if err != nil {
		return syncTransport{}, err
	}

	return newSyncTransport(t.sock, t.readTimeout), nil
}

func _send(writer io.Writer, msg []byte) error {
	for totalSent := 0; totalSent < len(msg); {
		sent, err := writer.Write(msg[totalSent:])
		if err != nil {
			return err
		}
		if sent == 0 {
			return ErrConnBroken
		}

		totalSent += sent
	}
	return nil
}

func _readN(reader io.Reader, size int) ([]byte, error) {
	raw := make([]byte, 0, size)
	for len(raw) < size {
		buf := make([]byte, size-len(raw))
		n, err := io.ReadFull(reader, buf)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, ErrConnBroken
		}

		raw = append(raw, buf...)
	}
	return raw, nil
}
