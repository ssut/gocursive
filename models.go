package main

import "net/url"

// ClientConfig holds the data from CLI main
type ClientConfig struct {
	url          *url.URL
	concurrent   int
	outputDir    string
	skipExisting bool
}
