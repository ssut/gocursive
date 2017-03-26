package main

import "io"

type ProxyReader struct {
	io.Reader
	total int64 // Total # of bytes transferred
}

func (reader *ProxyReader) Read(p []byte) (int, error) {
	n, err := reader.Reader.Read(p)
	reader.total += int64(n)

	return n, err
}

func (reader *ProxyReader) Total() int64 {
	return reader.total
}
