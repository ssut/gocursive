package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	humanize "github.com/dustin/go-humanize"
	"github.com/valyala/fasthttp"
	"golang.org/x/sys/unix"
)

// Client type is used for internal values
type Client struct {
	config      *ClientConfig
	httpClient  *fasthttp.Client
	directories []string
	files       []*url.URL
	bytesTotal  int64
	bytesRecv   int64
}

// NewClient method creates a new Gocursive client
func NewClient(config *ClientConfig) *Client {
	var client *Client

	httpClient := &fasthttp.Client{
		Name:            "Gocursive",
		MaxConnsPerHost: 10240,
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
		bytesTotal:  0,
		bytesRecv:   0,
	}

	return client
}

func (c *Client) checkWritable() bool {
	return unix.Access(c.config.outputDir, unix.W_OK) == nil
}

func (c *Client) collectUrls(target *url.URL, sem chan bool) {
	sem <- true
	defer func() { <-sem }()
	statusCode, body, err := c.httpClient.Get(nil, target.String())
	if err != nil {
		log.Panic(err)
	}
	if statusCode != fasthttp.StatusOK {
		log.Errorf("Unexpected status code: %d (from %s)", statusCode, target.String())
		return
	}
	hasIndexOf := strings.Contains(string(body), "Index of ")
	log.WithFields(logrus.Fields{
		"statusCode": statusCode,
		"hasIndexOf": hasIndexOf,
	}).Debugf("Hit: %s", target.Path)
	if !hasIndexOf {
		return
	}

	dirs, files := getURLs(target.String(), body)
	for _, dir := range dirs {
		c.directories = append(c.directories, dir)
		next := &url.URL{
			Scheme:   target.Scheme,
			Host:     target.Host,
			Path:     dir,
			RawQuery: target.RawQuery,
		}
		go c.collectUrls(next, sem)
	}
	for _, file := range files {
		c.files = append(c.files, file)
	}
}

func (c *Client) collectUrlsFromIndex() {
	sem := make(chan bool, c.config.concurrent)
	c.collectUrls(c.config.url, sem)

	for i := 0; i < cap(sem); i++ {
		sem <- true
	}
}

func (c *Client) createDirectories() {
	// if no directories found but have files to download
	if len(c.directories) == 0 && len(c.files) > 0 {
		path := filepath.Join(c.config.outputDir, filepath.Dir(c.files[0].Path))
		os.MkdirAll(path, 0755)
	}

	for _, dir := range c.directories {
		path := filepath.Join(c.config.outputDir, dir)
		os.MkdirAll(path, 0755)
	}
}

func (c *Client) download(filepath string, url *url.URL, sem <-chan bool) (err error) {
	defer func() { <-sem }()
	// fasthttp currently does not support streaming for response content
	resp, err := http.Get(url.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	size := resp.ContentLength
	if c.config.skipExisting {
		if f, err := os.Stat(filepath); err == nil {
			if size == f.Size() {
				log.WithFields(logrus.Fields{
					"size": humanize.Bytes(uint64(size)),
				}).Debugf("File exists: %s", url.Path)
				return nil
			}
		}
	}

	c.bytesTotal += size

	reader := &ProxyReader{Reader: resp.Body}
	go func() {
		var current int64
		for i := reader.Total(); i < size; i = reader.Total() {
			current = i - current
			c.bytesRecv += current
			current = i
			time.Sleep(500 * time.Millisecond)
		}
		current = reader.Total() - current
		c.bytesRecv += current
	}()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	started := time.Now()
	_, err = io.Copy(out, reader)
	if err != nil {
		return err
	}

	elapsed := time.Since(started)
	log.WithFields(logrus.Fields{
		"elapsed": elapsed.String(),
		"size":    humanize.Bytes(uint64(size)),
	}).Debugf("Downloaded: %s", url.Path)

	return nil
}

func (c *Client) start() {
	var current int64

	// this channel works as like a semaphore
	sem := make(chan bool, c.config.concurrent)
	go func() {
		var diff int64
		total := len(c.files)
		for {
			diff = c.bytesRecv - diff
			log.WithFields(logrus.Fields{
				"speed":   fmt.Sprintf("%s/s", humanize.Bytes(uint64(diff))),
				"running": len(sem),
			}).Infof("[%d/%d] %s/%s", current, total, humanize.Bytes(uint64(c.bytesRecv)), humanize.Bytes(uint64(c.bytesTotal)))
			diff = c.bytesRecv
			time.Sleep(time.Second)
		}
	}()

	for _, url := range c.files {
		sem <- true
		current++
		path, _ := filepath.Abs(filepath.Join(c.config.outputDir, url.Path))
		go c.download(path, url, sem)
	}

	for i := 0; i < cap(sem); i++ {
		sem <- true
	}
}

// Run method starts job
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
	log.Info("Done.")
}
