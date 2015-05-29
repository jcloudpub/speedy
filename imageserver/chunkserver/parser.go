package chunkserver

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type Resp struct {
	Type int8
	Len  int32
	Data []byte
}

const (
	HEADERSIZE = 6
)

func ReadHeader(r *bufio.Reader) (*Resp, error) {
	var header []byte = make([]byte, HEADERSIZE)

	_, err := io.ReadFull(r, header)
	if err != nil {
		return nil, err
	}

	typeBuf := bytes.NewBuffer(header[0:1])
	lenBuf := bytes.NewBuffer(header[1:HEADERSIZE])

	var t int8
	var len int32

	err = binary.Read(typeBuf, binary.BigEndian, &t)
	if err != nil {
		return nil, err
	}

	err = binary.Read(lenBuf, binary.BigEndian, &len)
	if err != nil {
		return nil, err
	}

	var data []byte = make([]byte, len)
	if data == nil {
		return nil, fmt.Errorf("malloc %s B error", len)
	}

	resp := &Resp{
		Type: t,
		Len:  len,
		Data: data,
	}

	return resp, nil
}

func Parse(r *bufio.Reader) (*Resp, error) {
	resp, err := ReadHeader(r)
	if err != nil {
		return nil, err
	}

	_, err = io.ReadFull(r, resp.Data)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (r *Resp) Bytes() ([]byte, error) {
	return nil, nil
}
