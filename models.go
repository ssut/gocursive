package main

import "net/url"

type ClientConfig struct {
	url        *url.URL
	concurrent int
	outputDir  string
}
