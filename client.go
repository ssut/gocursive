package main

import (
	"strings"

	"github.com/valyala/fasthttp"
)

type Client struct {
	config      *ClientConfig
	httpClient  *fasthttp.Client
	directories []string
	files       []string
}

func NewClient(config *ClientConfig) *Client {
	var client *Client

	httpClient := &fasthttp.Client{
		Name:            "Gocursive",
		MaxConnsPerHost: config.concurrent,
	}

	if !strings.HasSuffix(config.url, "/") {
		config.url += "/"
	}

	client = &Client{
		config:      config,
		httpClient:  httpClient,
		directories: []string{},
		files:       []string{},
	}

	return client
}

func (c *Client) collectUrls(url string) {
	statusCode, body, err := c.httpClient.Get(nil, url)
	if err != nil {
		log.Panic(err)
	}
	if statusCode != fasthttp.StatusOK {
		log.Fatalf("Unexpected status code: %d", statusCode)
	}

	_, _ = getURLs(url, body)
}

func (c *Client) collectUrlsFromIndex() {
	c.collectUrls(c.config.url)
}

func (c *Client) Run() {
	log.Info("Starting collecting URLs..")
	c.collectUrlsFromIndex()
}
