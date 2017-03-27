package main

import "io"

// ProxyReader type is used to know received bytes
type ProxyReader struct {
	io.Reader
	total int64 // Total # of bytes transferred
}

func (reader *ProxyReader) Read(p []byte) (int, error) {
	n, err := reader.Reader.Read(p)
	reader.total += int64(n)

	return n, err
}

// Total method returns the number of bytes received
func (reader *ProxyReader) Total() int64 {
	return reader.total
}
