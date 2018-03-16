package ngxnet

import (
	"bytes"
	"compress/zlib"
	"io"
)

func ZlibCompress(data []byte) []byte {
	var in bytes.Buffer
	w := zlib.NewWriter(&in)
	w.Write(data)
	w.Close()
	return in.Bytes()
}

func ZlibUnCompress(data []byte) []byte {
	b := bytes.NewReader(data)
	var out bytes.Buffer
	r, _ := zlib.NewReader(b)
	io.Copy(&out, r)
	return out.Bytes()
}
