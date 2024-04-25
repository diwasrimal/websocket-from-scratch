package myws

import (
	"io"
	"net"
)

// Stores bytes that are parsed from the websocket frame
type WsByteFrame struct {
	Final, Rsv1, Rsv2, Rsv3 byte
	Opcode                  byte
	Masked                  byte
	PayloadInitialLen       byte
	PayloadExtendedLen      []byte
	MaskingKey              []byte
	Payload                 []byte
}

const (
	OpcodeContinuation = 0x0
	OpcodeText         = 0x1
	OpcodeBinary       = 0x2
	// 0x3 - x07 reserved for futher non-control frames
	OpcodeClose = 0x8
	OpcodePing  = 0x9
	OpcodePong  = 0xa
	// 0xb - 0xf reserved for futher control frames
)

func (f *WsByteFrame) IsFinal() bool {
	return f.Final == 0b10000000
}

func ParseWsBytes(conn net.Conn) (WsByteFrame, error) {
	var bf WsByteFrame

	// Parse first byte
	first := make([]byte, 1)
	_, err := io.ReadFull(conn, first)
	if err != nil {
		return bf, err
	}
	bf.Final = first[0] & 0b10000000
	bf.Rsv1 = first[0] & 0b01000000
	bf.Rsv2 = first[0] & 0b00100000
	bf.Rsv3 = first[0] & 0b00010000
	bf.Opcode = first[0] & 0b00001111

	// Second byte
	second := make([]byte, 1)
	_, err = io.ReadFull(conn, second)
	if err != nil {
		return bf, err
	}
	bf.Masked = second[0] & 0b10000000
	bf.PayloadInitialLen = second[0] & 0b01111111

	payloadInitialLen := uint64(bf.PayloadInitialLen)

	extendedSize := 0
	if payloadInitialLen == 126 {
		extendedSize = 2
	} else if payloadInitialLen == 127 {
		extendedSize = 8
	}

	bf.PayloadExtendedLen = make([]byte, extendedSize)
	_, err = io.ReadFull(conn, bf.PayloadExtendedLen)
	if err != nil {
		return bf, err
	}

	var payloadLen uint64
	switch extendedSize {
	case 0:
		payloadLen = payloadInitialLen
	case 2:
		payloadLen = (uint64(bf.PayloadExtendedLen[0]) << 8) | uint64(bf.PayloadExtendedLen[1])
	case 8:
		payloadLen = uint64(bf.PayloadExtendedLen[0])<<56 |
			uint64(bf.PayloadExtendedLen[1])<<48 |
			uint64(bf.PayloadExtendedLen[2])<<40 |
			uint64(bf.PayloadExtendedLen[3])<<32 |
			uint64(bf.PayloadExtendedLen[4])<<24 |
			uint64(bf.PayloadExtendedLen[5])<<16 |
			uint64(bf.PayloadExtendedLen[6])<<8 |
			uint64(bf.PayloadExtendedLen[7])
	}

	var maskSize = 0
	isMasked := bf.Masked == 0b10000000
	if isMasked {
		maskSize = 4
	}
	bf.MaskingKey = make([]byte, maskSize)
	_, err = io.ReadFull(conn, bf.MaskingKey)
	if err != nil {
		return bf, err
	}

	bf.Payload = make([]byte, payloadLen)
	_, err = io.ReadFull(conn, bf.Payload)
	if err != nil {
		return bf, err
	}

	return bf, nil
}

func SendWsByteFrame(conn net.Conn, bf WsByteFrame) (int, error) {
	first := bf.Final | bf.Rsv1 | bf.Rsv2 | bf.Rsv3 | bf.Opcode
	second := bf.Masked | bf.PayloadInitialLen

	bytes := []byte{first, second}
	bytes = append(bytes, bf.PayloadExtendedLen...)
	bytes = append(bytes, bf.MaskingKey...)
	bytes = append(bytes, bf.Payload...)
	return conn.Write(bytes)
}
