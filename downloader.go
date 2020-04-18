package logstf

import (
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type FileFormat string

const ZipFormat FileFormat = ".zip"
const JSONFormat FileFormat = ".json"

func Exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func LogCacheFile(logId int64, fileType FileFormat) string {
	cacheDirRange := int64(0)
	if logId >= 10000 {
		cacheDirRange = (logId / 1000) * 1000
	}
	fileName := fmt.Sprintf("logs_%d%s", logId, fileType)
	return path.Join(fmt.Sprintf("%d", cacheDirRange), fileName)
}

func GetLatestLogId() (int64, error) {
	// Get the data
	resp, err := http.Get("http://logs.tf/")
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("Failed to close http response body")
		}
	}()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	return parseLatestLogId(b), nil
}

// UpdateCache does a more shallow update cycle meant to watch the homepage for new logs.
// Use the Downloader for fetching the full database from logstf
func UpdateCache(baseDir string, lookBackSize int64) error {
	currentId, err := GetLatestLogId()
	if err != nil {
		return err
	}
	stopId := currentId - lookBackSize
	if lookBackSize <= 0 {
		stopId = 0
	}
	client := &http.Client{}
	for currentId > stopId {
		fetched := false
		fp := path.Join(baseDir, LogCacheFile(currentId, ZipFormat))
		if !Exists(fp) {
			if err := FetchLogFile(client, currentId, fp); err != nil {
				if err.Error() != "too many requests" {
					log.Errorf("Failed to fetch log: %d", currentId)
				}
				currentId--
				continue
			}
			fetched = true
		}
		fpJson := path.Join(baseDir, LogCacheFile(currentId, JSONFormat))
		if !Exists(fpJson) {
			if err := FetchAPIFile(currentId, fpJson); err != nil {
				log.WithError(err).Errorf("Failed to update api cache: %d", currentId)
				currentId--
				continue
			}
		}
		if fetched {
			log.Infof("Fetched log: %d", currentId)
		}
		currentId--
	}
	return nil
}

func parseLatestLogId(body []byte) int64 {
	rx := regexp.MustCompile(`<tr id="log_(\d+)">`)
	m := rx.FindAllStringSubmatch(string(body), 25)
	largestId := int64(0)
	for _, l := range m {
		logId, err := strconv.ParseInt(l[1], 10, 64)
		if err != nil {
			log.Warnf("Failed to parse logid: %s", l[1])
			continue
		}
		if logId > largestId {
			largestId = logId
		}
	}
	return largestId
}

func FetchLogFile(client *http.Client, logId int64, savePath string) error {
	url := fmt.Sprintf("https://logs.tf/logs/log_%d.log.zip", logId)
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("failed to close resp body (FetchLogFile)")
		}
	}()
	// Make the subdir if needed
	if !Exists(filepath.Dir(savePath)) {
		if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
			return err
		}
	}
	out, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := out.Close(); err != nil {
			log.Warnf("failed to close file (FetchLogFile)")
		}
	}()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func NewDownloader(penaltyIncrement int) *Downloader {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	return &Downloader{
		wg:               &sync.WaitGroup{},
		client:           &http.Client{Transport: tr},
		queue:            make(chan downloadRequest, 100),
		Overwrite:        false,
		penaltyIncrement: penaltyIncrement,
	}
}

type downloadRequest struct {
	Url  string
	Path string
}

type Downloader struct {
	Failures         int64
	Successes        int64
	wg               *sync.WaitGroup
	client           *http.Client
	queue            chan downloadRequest
	stopRequested    chan interface{}
	Overwrite        bool
	waitTime         time.Duration
	penaltyIncrement int
}

func (d *Downloader) AddRequest(url string, path string) {
	d.queue <- downloadRequest{
		Url:  url,
		Path: path,
	}
}

func (d *Downloader) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-d.queue:
			if err := d.fetch(req); err != nil {
				if err != ErrNotFound && err != ErrTooMany {
					log.WithError(err).Errorf("Failed to fetch: %s", req.Url)
				}
				atomic.AddInt64(&d.Failures, 1)
			} else {
				atomic.AddInt64(&d.Successes, 1)
				log.Debugf("Downloaded: %s -> %s", req.Url, req.Path)
			}
		}
	}
}

func (d *Downloader) Start(workers int, ctx context.Context) {
	for i := 0; i < workers; i++ {
		go d.worker(ctx)
	}
	log.Debugf("Started workers (%d)", workers)
}

func (d *Downloader) Stop() {
	close(d.queue)
	d.stopRequested <- true
}

func (d *Downloader) Wait() {
	d.wg.Wait()
}

func (d *Downloader) tooMany(request downloadRequest) {
	d.queue <- request
	d.waitTime += time.Millisecond * time.Duration(d.penaltyIncrement)
	log.Warnf("Wait time increased: %s", d.waitTime.String())
	time.Sleep(time.Second)
}

func init() {
	ErrNotFound = errors.New("not found")
	ErrTooMany = errors.New("too many requests")
	ErrBadStatus = errors.New("bad status")
}

var ErrTooMany error
var ErrNotFound error
var ErrBadStatus error

func (d *Downloader) fetch(request downloadRequest) error {
	if !d.Overwrite && Exists(request.Path) {
		log.Debugf("Skipped fetch: %s", request.Url)
		return nil
	}
	time.Sleep(d.waitTime)
	resp, err := d.client.Get(request.Url)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("failed to close resp body (FetchLogFile)")
		}
	}()
	if resp.StatusCode == http.StatusTooManyRequests {
		d.tooMany(request)
		return ErrTooMany
	}
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ErrBadStatus
	}
	// Make the subdir if needed
	if !Exists(filepath.Dir(request.Path)) {
		if err := os.MkdirAll(filepath.Dir(request.Path), 0755); err != nil {
			return err
		}
	}
	out, err := os.Create(request.Path)
	if err != nil {
		return err
	}
	defer func() {
		if err := out.Close(); err != nil {
			log.Warnf("failed to close file (fetch)")
		}
	}()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
