package main

import "encoding/binary"
import "net"

// Current protocol is v1.
const ProtoVersion uint32 = 1

func sendVersion(conn net.Conn, version uint32) error {
	return sendUint32(conn, version)
}

func recvVersion(conn net.Conn) (uint32, error) {
	return recvUint32(conn)
}

func sendBool(conn net.Conn, val bool) error {
	var b byte = 0
	if val {
		b = 1
	}

	_, err := conn.Write([]byte { b })
	return err
}

func recvBool(conn net.Conn) (bool, error) {
	b := make([]byte, 1)
	_, err := conn.Read(b)
	if b[0] == 0 {
		return false, err
	} else {
		return true, err
	}
}

func sendUint32(conn net.Conn, val uint32) error {
	return binary.Write(conn, binary.BigEndian, val)
}

func recvUint32(conn net.Conn) (uint32, error) {
	var val uint32
	err := binary.Read(conn, binary.BigEndian, &val)
	return val, err
}
