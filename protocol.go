package main

import "encoding/binary"
import "fmt"
import "io"
import "io/ioutil"
import "os"
import "time"

type Version int32

// Current protocol is v1.
const ProtoVersion Version = 1

// Arbitrary limits to avoid allocating absurd amounts of space.
const MaxFileSize int64 = 1024 * 1024 * 1024 * 32
const MaxStringLength int32 = 1024
const MaxTimeLength int32 = 16

// Message terminator, to help debug protocol issues.
const MessageTerminator int32 = 20741

type Message interface{}

// Message types.
type MessageType int32
const (
	MsgBool MessageType = iota
	MsgCommand
	MsgFile
	MsgFileInfo
	MsgFileOffer
	MsgFileRequest
	MsgInt32
	MsgInt64
	MsgOfferFile
	MsgString
	MsgTime
	MsgUint32
	MsgVersion
)

var MessageTypeNames = map[MessageType]string {
	MsgBool: "MsgBool",
	MsgCommand: "MsgCommand",
	MsgFile: "MsgFile",
	MsgFileInfo: "MsgFileInfo",
	MsgFileOffer: "MsgFileOffer",
	MsgFileRequest: "MsgFileRequest",
	MsgInt32: "MsgInt32",
	MsgInt64: "MsgInt64",
	MsgOfferFile: "MsgOfferFile",
	MsgString: "MsgString",
	MsgTime: "MsgTime",
	MsgUint32: "MsgUint32",
	MsgVersion: "MsgVersion",
}

// Enumeration of commands.
type Command int32
const (
	CmdRequestNextFileInfo Command = iota
)

type FileInfo struct {
	Path string
	IsDir bool
	Mode os.FileMode
	ModTime time.Time
	Size int64
}

type FileRequest struct {
	Path string
}

type FileOffer struct {
	Info FileInfo
}

// Writes a message to the connection.
func send(conn io.Writer, msg Message) (err error) {
	switch msg := msg.(type) {
	default:
		err = fmt.Errorf("Unexpected type: %T", msg)
	case bool:
		err = sendBool(conn, msg)
	case Command:
		err = sendCommand(conn, msg)
	case FileInfo:
		err = sendFileInfo(conn, msg)
	case FileOffer:
		err = sendFileOffer(conn, msg)
	case FileRequest:
		err = sendFileRequest(conn, msg)
	case int32:
		err = sendInt32(conn, msg)
	case int64:
		err = sendInt64(conn, msg)
	case string:
		err = sendString(conn, msg)
	case time.Time:
		err = sendTime(conn, msg)
	case uint32:
		err = sendUint32(conn, msg)
	case Version:
		err = sendVersion(conn, msg)
	}

	if err == nil {
		err = writeMessageTerminator(conn)
	}

	return
}

// Reads a message from the connection.
func recv(conn io.Reader) (msg Message, msgType MessageType, err error) {
	msgType, err = recvMessageType(conn)
	if err != nil {
		return
	}

	msg, err = read(conn, msgType)
	if err == nil {
		err = checkMessageTerminator(conn)
	}

	return
}

// Reads message data from the connection.
func read(conn io.Reader, msgType MessageType) (msg Message, err error) {
	switch msgType {
	default:
		if name, ok := MessageTypeNames[msgType]; ok {
			err = fmt.Errorf("Unexpected message type: %s", name)
		} else {
			err = fmt.Errorf("Unexpected message type: %d", msgType)
		}
	case MsgBool:
		msg, err = recvBool(conn)
	case MsgCommand:
		msg, err = recvCommand(conn)
	case MsgFileInfo:
		msg, err = recvFileInfo(conn)
	case MsgFileOffer:
		msg, err = recvFileOffer(conn)
	case MsgFileRequest:
		msg, err = recvFileRequest(conn)
	case MsgInt32:
		msg, err = recvInt32(conn)
	case MsgInt64:
		msg, err = recvInt64(conn)
	case MsgString:
		msg, err = recvString(conn)
	case MsgTime:
		msg, err = recvTime(conn)
	case MsgUint32:
		msg, err = recvUint32(conn)
	case MsgVersion:
		msg, err = recvVersion(conn)
	}

	return
}

// Reads a message from the connection, checking that it is the expected type.
func expect(conn io.Reader, mt MessageType) (msg Message, err error) {
	msgType, err := recvMessageType(conn)
	if err != nil {
		return
	}

	if msgType != mt {
		if name, ok := MessageTypeNames[msgType]; ok {
			err = fmt.Errorf("Expected message type %v, got %v", MessageTypeNames[mt], name)
		} else {
			err = fmt.Errorf("Expected message type %v, got unknown type: %v", MessageTypeNames[mt], msgType)
		}
		return
	}

	msg, err = read(conn, msgType)
	if err == nil {
		err = checkMessageTerminator(conn)
	}

	return
}

func checkMessageTerminator(conn io.Reader) (err error) {
	term, err := recvInt32(conn)

	if err == nil && term != MessageTerminator {
		err = fmt.Errorf("Expected message terminator (%d), got: %d", MessageTerminator, term)
	}

	return
}

// Send/receive definitions
// send: Sends the message type followed by the message data.
// write: Sends only the raw message data.
// recv: Assumes message type has already been read, reads only the message
// data. Validates that data matches the expected type/constraints.
// expect: Consumes a message type (asserting that it matches the expected
// type) and the message data, then checks the message terminator.

func sendBool(conn io.Writer, b bool) (err error) {
	err = writeMessageType(conn, MsgBool)
	if err != nil {
		return
	}
	if b {
		err = writeByte(conn, 1)
	} else {
		err = writeByte(conn, 0)
	}
	return
}

func recvBool(conn io.Reader) (b bool, err error) {
	bt, err := recvByte(conn)
	if err != nil {
		return
	}

	return bt != 0, nil
}

func expectBool(conn io.Reader) (b bool, err error) {
	msg, _, err := recv(conn)
	if err != nil {
		return
	}

	var ok bool
	if b, ok = msg.(bool); !ok {
		err = fmt.Errorf("Expected bool, got %T: %v", msg, msg)
	}

	return
}

func recvByte(conn io.Reader) (b byte, err error) {
	buf := make([]byte, 1)
	_, err = io.ReadFull(conn, buf)
	return buf[0], err
}

func writeByte(conn io.Writer, b byte) (err error) {
	_, err = conn.Write([]byte { b })
	return
}

func sendCommand(conn io.Writer, cmd Command) (err error) {
	err = writeMessageType(conn, MsgCommand)
	if err != nil {
		return
	}

	err = writeInt32(conn, int32(cmd))
	return
}

func recvCommand(conn io.Reader) (cmd Command, err error) {
	c, err := recvInt32(conn)
	return Command(c), err
}

func expectCommand(conn io.Reader) (cmd Command, err error) {
	msg, _, err := recv(conn)
	if err != nil {
		return
	}

	var ok bool
	if cmd, ok = msg.(Command); !ok {
		err = fmt.Errorf("Expected Command, got %T: %v", msg, msg)
	}

	return
}

func sendFile(conn io.Writer, fi FileInfo, path string) (err error) {
	err = writeMessageType(conn, MsgFile)
	if err != nil {
		return
	}

	file, err := os.Open(path)
	if err != nil {
		return
	}

	err = send(conn, fi)
	if err != nil {
		return
	}

	n, err := io.Copy(conn, file)
	if err != nil {
		return
	}
	if n != fi.Size {
		return fmt.Errorf("Only %d of %d bytes were sent for %s", n, fi.Size, fi.Path)
	}

	err = writeMessageTerminator(conn)
	return
}

func recvFile(conn io.Reader, expected FileInfo, targetPath string, overwrite bool) (err error) {
	if !overwrite {
		if _, err = os.Stat(targetPath); !os.IsNotExist(err) {
			err = fmt.Errorf("Refusing to overwrite %s.", targetPath)
			return
		}
	}

	err = expectMessageType(conn, MsgFile)
	if err != nil {
		return
	}

	fi, err := expectFileInfo(conn)
	if err != nil {
		return
	}

	if fi.Path != expected.Path {
		return fmt.Errorf("Requested %v, server sent %v.", expected.Path, fi.Path)
	}

	if fi.Size > MaxFileSize {
		return fmt.Errorf("File too large: %d bytes", fi.Size)
	}

	// File is saved to a temp file until fully received.
	temp, err := ioutil.TempFile("", "zync")
	if err != nil {
		return
	}

	written, err := io.CopyN(temp, conn, fi.Size)
	if err != nil {
		return
	}
	if written != fi.Size {
		return fmt.Errorf("Failed to receive full contents of %s (%d bytes)", expected.Path, fi.Size)
	}

	err = checkMessageTerminator(conn)
	if err != nil {
		return
	}

	// Move the temp file to the specified location.
	err = os.Rename(temp.Name(), targetPath)
	if err != nil {
		return
	}

	// Update the modtime of the file to match the provider's.
	err = os.Chtimes(targetPath, fi.ModTime, fi.ModTime)
	return
}

func sendFileInfo(conn io.Writer, fi FileInfo) (err error) {
	err = writeMessageType(conn, MsgFileInfo)
	if err != nil {
		return
	}

	err = send(conn, fi.Path)
	if err != nil {
		return
	}

	err = send(conn, fi.IsDir)
	if err != nil {
		return
	}

	err = send(conn, uint32(fi.Mode))
	if err != nil {
		return
	}

	err = send(conn, fi.ModTime)
	if err != nil {
		return
	}

	err = send(conn, fi.Size)
	return
}

func recvFileInfo(conn io.Reader) (fi FileInfo, err error) {
	path, err := expectString(conn)
	if err != nil {
		return
	}

	isDir, err := expectBool(conn)
	if err != nil {
		return
	}

	mode, err := expectUint32(conn)
	if err != nil {
		return
	}

	modTime, err := expectTime(conn)
	if err != nil {
		return
	}

	size, err := expectInt64(conn)
	if err != nil {
		return
	}

	fi.Path = path
	fi.IsDir = isDir
	fi.Mode = os.FileMode(mode)
	fi.ModTime = modTime
	fi.Size = size
	return
}

func expectFileInfo(conn io.Reader) (fi FileInfo, err error) {
	msg, _, err := recv(conn)
	if err != nil {
		return
	}

	var ok bool
	if fi, ok = msg.(FileInfo); !ok {
		err = fmt.Errorf("Expected FileInfo, got %T: %v", msg, msg)
	}

	return
}

func sendFileOffer(conn io.Writer, offer FileOffer) (err error) {
	err = writeMessageType(conn, MsgFileOffer)
	if err != nil {
		return
	}

	err = send(conn, offer.Info)
	return
}

func recvFileOffer(conn io.Reader) (offer FileOffer, err error) {
	info, err := expectFileInfo(conn)
	if err != nil {
		return
	}

	offer.Info = info
	return
}

func sendFileRequest(conn io.Writer, req FileRequest) (err error) {
	err = writeMessageType(conn, MsgFileRequest)
	if err != nil {
		return
	}

	err = send(conn, req.Path)
	return
}

func recvFileRequest(conn io.Reader) (req FileRequest, err error) {
	path, err := expectString(conn)
	if err != nil {
		return
	}

	req.Path = path
	return
}

func sendInt32(conn io.Writer, val int32) (err error) {
	err = writeMessageType(conn, MsgInt32)
	if err != nil {
		return
	}

	return writeInt32(conn, val)
}

func recvInt32(conn io.Reader) (val int32, err error) {
	err = binary.Read(conn, binary.BigEndian, &val)
	return
}

func writeInt32(conn io.Writer, val int32) (err error) {
	return binary.Write(conn, binary.BigEndian, val)
}

func sendInt64(conn io.Writer, val int64) (err error) {
	err = writeMessageType(conn, MsgInt64)
	if err != nil {
		return
	}

	return writeInt64(conn, val)
}

func recvInt64(conn io.Reader) (val int64, err error) {
	err = binary.Read(conn, binary.BigEndian, &val)
	return
}

func expectInt64(conn io.Reader) (val int64, err error) {
	msg, _, err := recv(conn)
	if err != nil {
		return
	}

	var ok bool
	if val, ok = msg.(int64); !ok {
		err = fmt.Errorf("Expected int64, got %T: %v", msg, msg)
	}

	return
}

func writeInt64(conn io.Writer, val int64) error {
	return binary.Write(conn, binary.BigEndian, val)
}

func writeMessageTerminator(conn io.Writer) error {
	return writeInt32(conn, MessageTerminator)
}

func recvMessageType(conn io.Reader) (mt MessageType, err error) {
	var msgType uint32
	err = binary.Read(conn, binary.BigEndian, &msgType)
	return MessageType(msgType), err
}

func expectMessageType(conn io.Reader, mt MessageType) (err error) {
	msgType, err := recvMessageType(conn)
	if err != nil {
		return
	}

	if msgType != mt {
		return fmt.Errorf("Expected message type %d, got %d.", mt, msgType)
	}

	return nil
}

func writeMessageType(conn io.Writer, mt MessageType) (err error) {
	return writeInt32(conn, int32(mt))
}

func sendString(conn io.Writer, s string) (err error) {
	err = writeMessageType(conn, MsgString)
	if err != nil {
		return
	}

	err = writeInt32(conn, int32(len(s)))
	if err != nil {
		return
	}

	_, err = conn.Write([]byte(s))
	return
}

func recvString(conn io.Reader) (s string, err error) {
	length, err := recvInt32(conn)
	if err != nil {
		return
	}
	if length > MaxStringLength {
		err = fmt.Errorf("String of length %d exceeds max of %d", length, MaxStringLength)
		return
	}

	buffer := make([]byte, length)
	_, err = io.ReadFull(conn, buffer)
	return string(buffer), err
}

func expectString(conn io.Reader) (s string, err error) {
	msg, _, err := recv(conn)
	if err != nil {
		return
	}

	var ok bool
	if s, ok = msg.(string); !ok {
		err = fmt.Errorf("Expected string, got %T: %v", msg, msg)
	}

	return
}

func sendTime(conn io.Writer, t time.Time) (err error) {
	err = writeMessageType(conn, MsgTime)
	if err != nil {
		return
	}

	buf, err := t.MarshalBinary()
	if err != nil {
		return
	}

	err = writeInt32(conn, int32(len(buf)))
	if err != nil {
		return
	}

	_, err = conn.Write(buf)
	return
}

func recvTime(conn io.Reader) (t time.Time, err error) {
	length, err := recvInt32(conn)
	if err != nil {
		return
	}
	if length > MaxTimeLength {
		err = fmt.Errorf("Time of length %d exceeds max of %d", length, MaxTimeLength)
		return
	}

	buf := make([]byte, length)
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return
	}

	err = t.UnmarshalBinary(buf)
	return
}

func expectTime(conn io.Reader) (t time.Time, err error) {
	msg, _, err := recv(conn)
	if err != nil {
		return
	}

	var ok bool
	if t, ok = msg.(time.Time); !ok {
		err = fmt.Errorf("Expected time, got %T: %v", msg, msg)
	}

	return
}

func sendUint32(conn io.Writer, val uint32) (err error) {
	err = writeMessageType(conn, MsgUint32)
	if err != nil {
		return
	}

	return writeUint32(conn, val)
}

func recvUint32(conn io.Reader) (val uint32, err error) {
	err = binary.Read(conn, binary.BigEndian, &val)
	return
}

func writeUint32(conn io.Writer, val uint32) (err error) {
	return binary.Write(conn, binary.BigEndian, val)
}

func expectUint32(conn io.Reader) (val uint32, err error) {
	msg, _, err := recv(conn)
	if err != nil {
		return
	}

	var ok bool
	if val, ok = msg.(uint32); !ok {
		err = fmt.Errorf("Expected uint32, got %T: %v", msg, msg)
	}

	return
}

func sendVersion(conn io.Writer, v Version) (err error) {
	err = writeMessageType(conn, MsgVersion)
	if err != nil {
		return
	}

	err = writeInt32(conn, int32(v))
	return
}

func recvVersion(conn io.Reader) (v Version, err error) {
	ver, err := recvInt32(conn)
	return Version(ver), err
}

func expectVersion(conn io.Reader) (v Version, err error) {
	msg, _, err := recv(conn)
	if err != nil {
		return
	}

	var ok bool
	if v, ok = msg.(Version); !ok {
		err = fmt.Errorf("Expected Version, got %T: %v", msg, msg)
	}

	return
}
