package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cheggaaa/pb"
	"github.com/valyala/fasthttp"
	"golang.org/x/sys/unix"
)

type Client struct {
	config      *ClientConfig
	httpClient  *fasthttp.Client
	directories []string
	files       []*url.URL
}

func NewClient(config *ClientConfig) *Client {
	var client *Client

	httpClient := &fasthttp.Client{
		Name: "Gocursive",
	}

	if !strings.HasSuffix(config.url.Path, "/") {
		config.url.Path += "/"
	}

	config.outputDir, _ = filepath.Abs(config.outputDir)

	client = &Client{
		config:      config,
		httpClient:  httpClient,
		directories: []string{},
		files:       []*url.URL{},
	}

	return client
}

func (c *Client) checkWritable() bool {
	return unix.Access(c.config.outputDir, unix.W_OK) == nil
}

func (c *Client) collectUrls(target *url.URL, wg *sync.WaitGroup) {
	statusCode, body, err := c.httpClient.Get(nil, target.String())
	if err != nil {
		log.Panic(err)
	}
	if statusCode != fasthttp.StatusOK {
		log.Fatalf("Unexpected status code: %d", statusCode)
	}
	log.Debugf("Hit: %s", target.Path)

	dirs, files := getURLs(target.String(), body)
	for _, dir := range dirs {
		c.directories = append(c.directories, dir)
		wg.Add(1)
		next := &url.URL{
			Scheme:   target.Scheme,
			Host:     target.Host,
			Path:     dir,
			RawQuery: target.RawQuery,
		}
		go c.collectUrls(next, wg)
	}
	for _, file := range files {
		c.files = append(c.files, file)
	}

	wg.Done()
}

func (c *Client) collectUrlsFromIndex() {
	var wg sync.WaitGroup
	wg.Add(1)
	go c.collectUrls(c.config.url, &wg)
	wg.Wait()
}

func (c *Client) createDirectories() {
	for _, dir := range c.directories {
		path := filepath.Join(c.config.outputDir, dir)
		os.MkdirAll(path, 0755)
	}
}

func (c *Client) download(filepath string, url *url.URL, sem <-chan bool) (err error) {
	defer func() { <-sem }()
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// fasthttp currently does not support streaming for response content
	resp, err := http.Get(url.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// progressbar
	prefix := fmt.Sprintf("%s", url.Path)
	bar := pb.New(int(resp.ContentLength)).SetUnits(pb.U_BYTES).Prefix(prefix)
	bar.Start()
	defer bar.Finish()

	reader := bar.NewProxyReader(resp.Body)
	_, err = io.Copy(out, reader)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) start() {

	sem := make(chan bool, c.config.concurrent)
	for _, url := range c.files {
		sem <- true
		path, _ := filepath.Abs(filepath.Join(c.config.outputDir, url.Path))
		go c.download(path, url, sem)
	}

	for i := 0; i < cap(sem); i++ {
		sem <- true
	}
}

func (c *Client) Run() {
	log.Infof("Checking permission for: %s", c.config.outputDir)
	if !c.checkWritable() {
		log.Panicf("No write permission on directory: %s", c.config.outputDir)
		os.Exit(1)
	}

	log.Info("Starting collecting URLs..")
	c.collectUrlsFromIndex()
	log.Infof("Total files found: %d, Total directories found: %d", len(c.files), len(c.directories))
	log.Info("Creating the same directory structure..")
	c.createDirectories()
	log.Info("Preparing for downloads..")
	c.start()
}
