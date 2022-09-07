package gadb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

const (
	syncMaxChunkSize = 64 * 1024
)

type syncTransport struct {
	sock        net.Conn
	readTimeout time.Duration
}

func newSyncTransport(sock net.Conn, readTimeout time.Duration) syncTransport {
	return syncTransport{
		sock:        sock,
		readTimeout: readTimeout,
	}
}

func (sync syncTransport) Send(command, data string) error {
	if len(command) != 4 {
		return errors.New("sync commands must have length 4")
	}

	msg := bytes.NewBufferString(command)
	err := binary.Write(msg, binary.LittleEndian, int32(len(data)))
	if err != nil {
		return fmt.Errorf("sync transport write: %w", err)
	}
	msg.WriteString(data)

	err = _send(sync.sock, msg.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (sync syncTransport) SendStream(reader io.Reader) error {
	for {
		b := make([]byte, syncMaxChunkSize)

		n, err := reader.Read(b)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		err = sync.sendChunk(b[:n])
		if err != nil {
			return err
		}
	}
}

func (sync syncTransport) SendStatus(statusCode string, n uint32) error {
	msg := bytes.NewBufferString(statusCode)
	err := binary.Write(msg, binary.LittleEndian, n)
	if err != nil {
		return fmt.Errorf("sync transport write: %w", err)
	}

	err = _send(sync.sock, msg.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (sync syncTransport) sendChunk(buffer []byte) error {
	msg := bytes.NewBufferString("DATA")
	err := binary.Write(msg, binary.LittleEndian, int32(len(buffer)))
	if err != nil {
		return fmt.Errorf("sync transport write: %w", err)
	}

	msg.Write(buffer)
	err = _send(sync.sock, msg.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (sync syncTransport) VerifyStatus() error {
	status, err := sync.ReadStringN(4)
	if err != nil {
		return err
	}

	tmpUint32, err := sync.ReadUint32()
	if err != nil {
		return fmt.Errorf("sync transport read (status): %w", err)
	}

	msg, err := sync.ReadStringN(int(tmpUint32))
	if err != nil {
		return err
	}

	if status == "FAIL" {
		return fmt.Errorf("sync verify status (fail): %s", msg)
	}

	if status != "OKAY" {
		return fmt.Errorf("sync verify status: Unknown error: %s", msg)
	}

	return nil
}

func (sync syncTransport) WriteStream(dest io.Writer) error {
	for {
		chunk, err := sync.readChunk()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("sync read chunk: %w", err)
		}

		err = _send(dest, chunk)
		if err != nil {
			return fmt.Errorf("sync write stream: %w", err)
		}
	}
}

func (sync syncTransport) readChunk() ([]byte, error) {
	status, err := sync.ReadStringN(4)
	if err != nil {
		return nil, err
	}

	tmpUint32, err := sync.ReadUint32()
	if err != nil {
		return nil, fmt.Errorf("read chunk (length): %w", err)
	}

	switch status {
	case "FAIL":
		sError, err := sync.ReadStringN(int(tmpUint32))
		if err != nil {
			return nil, fmt.Errorf("read chunk (error message): %w", err)
		}
		return nil, fmt.Errorf("status (fail): %s", sError)

	case "DONE":
		return nil, io.EOF

	case "DATA":
		chunk, err := sync.ReadBytesN(int(tmpUint32))
		if err != nil {
			return nil, err
		}
		return chunk, nil

	default:
		return nil, fmt.Errorf("unknown error: %q", status)
	}
}

func (sync syncTransport) ReadDirectoryEntry() (os.FileInfo, bool, error) {
	status, err := sync.ReadStringN(4)
	if err != nil {
		return fileInfo{}, false, err
	}
	if status == "DONE" {
		return fileInfo{}, false, nil
	}

	var entry fileInfo

	err = binary.Read(sync.sock, binary.LittleEndian, &entry.mode)
	if err != nil {
		return fileInfo{}, false, fmt.Errorf("sync transport read (mode): %w", err)
	}

	entry.size, err = sync.ReadUint32()
	if err != nil {
		return fileInfo{}, false, fmt.Errorf("sync transport read (size): %w", err)
	}

	lastModUnix, err := sync.ReadUint32()
	if err != nil {
		return fileInfo{}, false, fmt.Errorf("sync transport read (time): %w", err)
	}

	entry.modTime = time.Unix(int64(lastModUnix), 0)

	fLen, err := sync.ReadUint32()
	if err != nil {
		return fileInfo{}, false, fmt.Errorf("sync transport read (file name length): %w", err)
	}

	entry.name, err = sync.ReadStringN(int(fLen))
	if err != nil {
		return fileInfo{}, false, fmt.Errorf("sync transport read (file name): %w", err)
	}

	return entry, true, nil
}

func (sync syncTransport) ReadUint32() (uint32, error) {
	var n uint32
	err := binary.Read(sync.sock, binary.LittleEndian, &n)
	return n, err
}

func (sync syncTransport) ReadStringN(size int) (string, error) {
	raw, err := sync.ReadBytesN(size)
	if err != nil {
		return "", err
	}

	return string(raw), nil
}

func (sync syncTransport) ReadBytesN(size int) ([]byte, error) {
	_ = sync.sock.SetReadDeadline(time.Now().Add(sync.readTimeout))
	return _readN(sync.sock, size)
}

func (sync syncTransport) Close() error {
	if sync.sock == nil {
		return nil
	}
	return sync.sock.Close()
}
