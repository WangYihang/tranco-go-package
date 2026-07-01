//go:generate go run tool/version/generate.go
package tranco

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WangYihang/tranco-go-package/pkg/version"
	"github.com/schollz/progressbar/v3"
)

// defaultHTTPTimeout bounds how long a single HTTP request (list ID lookup
// or list download) may take before it is aborted, so a stalled connection
// cannot block callers forever.
const defaultHTTPTimeout = 5 * time.Minute

// defaultBaseURL is the Tranco List origin used whenever TrancoList.baseURL
// is unset, which is always the case for lists built via NewTrancoList.
// Tests construct a TrancoList directly and set baseURL to a local
// httptest server instead of talking to the real API.
const defaultBaseURL = "https://tranco-list.eu"

type TrancoList struct {
	ID               string
	Date             string
	IncludeSubdomain bool
	Scale            string
	CacheFolder      string
	cache            map[string]int64
	loaded           bool
	cacheMu          sync.Mutex
	httpClient       *http.Client
	userAgent        string
	baseURL          string
}

func (t *TrancoList) resolvedBaseURL() string {
	if t.baseURL != "" {
		return t.baseURL
	}
	return defaultBaseURL
}

func NewTrancoList(date string, includeSubdomain bool, scale string, cacheFolder string) (*TrancoList, error) {
	slog.Debug("obtaining tranco list id", slog.String("date", date), slog.Bool("includeSubdomain", includeSubdomain), slog.String("scale", scale))
	list := TrancoList{
		Date:             date,
		IncludeSubdomain: includeSubdomain,
		Scale:            scale,
		httpClient:       &http.Client{Timeout: defaultHTTPTimeout},
		userAgent:        fmt.Sprintf("%s Go-http-client/1.1 tranco-go/%s", strings.Replace(runtime.Version(), "go", "go/", 1), version.PV.Version),
		CacheFolder:      cacheFolder,
	}
	listID, err := list.getTrancoListID(date, includeSubdomain)
	if err != nil {
		return nil, err
	}
	list.ID = listID
	slog.Debug("downloading tranco list", slog.String("id", listID))
	filePath, err := list.DefaultFilePath()
	if err != nil {
		return nil, err
	}
	err = list.Download(filePath)
	if err != nil {
		slog.Error("error occurs when downloading tranco list", slog.String("id", listID), slog.String("error", err.Error()))
		return nil, err
	}
	slog.Debug("tranco list downloaded", slog.String("id", listID))
	return &list, nil
}

func (t *TrancoList) URL() string {
	return fmt.Sprintf("%s/download/%s/%s", t.resolvedBaseURL(), t.ID, t.Scale)
}

func (t *TrancoList) Rank(domain string) (int64, error) {
	// Rank() may be called concurrently for the same *TrancoList (e.g. from
	// tranco-server's HTTP handlers), and the cache map below is not safe
	// for concurrent read/write without this lock.
	t.cacheMu.Lock()
	defer t.cacheMu.Unlock()

	// load from cache
	if t.cache == nil {
		t.cache = make(map[string]int64)
	}

	if rank, ok := t.cache[domain]; ok {
		return rank, nil
	}

	// The whole list has already been scanned once and domain wasn't in it,
	// so there's no need to re-scan the (potentially multi-million-line)
	// file again just to reach the same conclusion.
	if t.loaded {
		return 0, fmt.Errorf("domain %s not found in tranco list", domain)
	}

	filePath, err := t.DefaultFilePath()
	if err != nil {
		return 0, err
	}

	fd, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer fd.Close()

	scanner := bufio.NewScanner(fd)
	slog.Debug("scanning tranco list", slog.String("filepath", filePath))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		currentRank, currentDomain, err := parseLine(line)
		if err != nil {
			slog.Warn("skipping malformed tranco list line", slog.String("error", err.Error()))
			continue
		}
		slog.Debug("scanning tranco list", slog.String("domain", currentDomain), slog.Int64("rank", currentRank))
		t.cache[currentDomain] = currentRank
		if currentDomain == domain {
			return currentRank, nil
		}
	}
	t.loaded = true

	return 0, fmt.Errorf("domain %s not found in tranco list", domain)
}

func (t *TrancoList) DefaultFilePath() (string, error) {
	var listType string
	if t.IncludeSubdomain {
		listType = "fqdn"
	} else {
		listType = "sld"
	}

	folder := t.CacheFolder
	if folder == "" {
		folder = ".tranco"
	}
	if !filepath.IsAbs(folder) {
		baseFolder, err := os.UserHomeDir()
		if err != nil {
			baseFolder = os.TempDir()
		}
		folder = filepath.Join(baseFolder, folder)
	}

	if err := os.MkdirAll(folder, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache folder %q: %w", folder, err)
	}

	filename := fmt.Sprintf("%s_%s_%s_%s.csv", t.Date, listType, t.Scale, t.ID)
	return filepath.Join(folder, filename), nil
}

func (t *TrancoList) newHTTPGetRequest(url string) (*http.Request, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("error occurs when creating HTTP request", slog.String("url", url), slog.String("error", err.Error()))
		return nil, err
	}
	request.Header.Set("User-Agent", t.userAgent)
	return request, nil
}

func (t *TrancoList) Download(filePath string) error {
	if _, err := os.Stat(filePath); err == nil {
		return nil
	}

	url := t.URL()

	slog.Info("downloading", slog.String("from", url), slog.String("to", filePath))

	request, err := t.newHTTPGetRequest(url)
	if err != nil {
		return err
	}

	response, err := t.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status code %d when downloading %s", response.StatusCode, url)
	}

	// Download to a temporary file first and rename on success, so a failed
	// or interrupted download never leaves a partial file at filePath (which
	// would otherwise be mistaken for a complete, cached download later on).
	tmpFilePath := filePath + ".tmp"
	fd, err := os.OpenFile(tmpFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	bar := progressbar.DefaultBytes(
		response.ContentLength,
		"downloading",
	)

	_, copyErr := io.Copy(io.MultiWriter(fd, bar), response.Body)
	closeErr := fd.Close()

	if copyErr != nil {
		os.Remove(tmpFilePath)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tmpFilePath)
		return closeErr
	}

	if err := os.Rename(tmpFilePath, filePath); err != nil {
		return err
	}

	slog.Info("downloaded", slog.String("filepath", filePath))
	return nil
}

func (t *TrancoList) getTrancoListID(date string, subdomain bool) (string, error) {
	urlObject, err := url.Parse(t.resolvedBaseURL())
	if err != nil {
		return "", err
	}
	urlObject.Path = "daily_list_id"
	query := urlObject.Query()
	query.Set("date", date)
	query.Set("subdomains", strconv.FormatBool(subdomain))
	urlObject.RawQuery = query.Encode()

	request, err := t.newHTTPGetRequest(urlObject.String())
	if err != nil {
		return "", err
	}

	response, err := t.httpClient.Do(request)
	if err != nil {
		slog.Error("error occurs when sending HTTP request", slog.String("url", urlObject.String()), slog.String("error", err.Error()))
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		slog.Error("error occurs when sending HTTP request", slog.String("url", urlObject.String()), slog.Int("statusCode", response.StatusCode))
		return "", fmt.Errorf("HTTP status code %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		slog.Error("error occurs when reading HTTP response body", slog.String("url", urlObject.String()), slog.String("error", err.Error()))
		return "", err
	}

	if bytes.Equal(body, []byte("null")) {
		slog.Error("no list id, api returns null", slog.String("date", date))
		return "", fmt.Errorf("no list id for %s, api returns null", date)
	}

	if bytes.Equal(body, []byte("500 Internal Server Error")) {
		slog.Error("no list id, api returns 500 Internal Server Error", slog.String("date", date))
		return "", fmt.Errorf("no list id for %s, api returns 500 Internal Server Error", date)
	}

	return string(body), nil
}

func Version() string {
	return version.Tag
}

func parseLine(line string) (int64, string, error) {
	parts := strings.Split(line, ",")
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("malformed tranco list line %q: expected 2 comma-separated fields, got %d", line, len(parts))
	}

	domain := parts[1]
	rank, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("malformed tranco list line %q: %w", line, err)
	}

	return rank, domain, nil
}
