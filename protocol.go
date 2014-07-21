package main

import "encoding/binary"
import "fmt"
import "io"

// Current protocol is v1.
const ProtoVersion uint32 = 1

// Arbitrary limit to avoid allocating absurd buffer space.
const MaxFilenameLength uint32 = 1024

const (
  RequestNextFileInfo uint32 = iota
)

func sendVersion(conn io.Writer, version uint32) error {
	return sendUint32(conn, version)
}

func recvVersion(conn io.Reader) (uint32, error) {
	return recvUint32(conn)
}

func sendBool(conn io.Writer, val bool) error {
	var b byte = 0
	if val {
		b = 1
	}

	_, err := conn.Write([]byte { b })
	return err
}

func recvBool(conn io.Reader) (bool, error) {
	b := make([]byte, 1)
	_, err := conn.Read(b)
	if b[0] == 0 {
		return false, err
	} else {
		return true, err
	}
}

func sendUint32(conn io.Writer, val uint32) error {
	return binary.Write(conn, binary.BigEndian, val)
}

func recvUint32(conn io.Reader) (uint32, error) {
	var val uint32
	err := binary.Read(conn, binary.BigEndian, &val)
	return val, err
}

// Writes a filename to the connection.
func sendFilename(conn io.Writer, fname string) error {
  // A filename is sent as a uint32 byte length, followed by the bytes of the
  // string itself.
  err := sendUint32(conn, uint32(len(fname)))
  if err != nil {
    return err
  }

  _, err = conn.Write([]byte(fname))
  return err
}


var fnameBuffer = make([]byte, MaxFilenameLength)

// Reads a filename from the connection.
func recvFilename(conn io.Reader) (string, error) {
  length, err := recvUint32(conn)
  if err != nil {
    return "", err
  }
  if length > MaxFilenameLength {
    return "", fmt.Errorf("Filename length %d exceed max buffer size %d.", length, MaxFilenameLength)
  }

  err = recvFully(conn, fnameBuffer, length)
  if err != nil {
    return "", err
  }

  return string(fnameBuffer[:length]), nil
}

// Attempts to fill the provided buffer with data read from the connection.
func recvFully(conn io.Reader, buffer []byte, length uint32) error {
  if length > uint32(len(buffer)) {
    panic(fmt.Errorf("Cannot read %d bytes into buffer of size %d.", length, len(buffer)))
  }

  var count uint32 = 0
  for count < length {
    c, err := conn.Read(buffer[count:])
    if err != nil {
      return err
    }
    count += uint32(c)
  }

  return nil
}
