//go:generate go run tool/version/generate.go

// Package tranco provides a Go client for the Tranco List
// (https://tranco-list.eu/), a research-oriented top-sites ranking. It looks
// up the list ID for a given date, downloads and caches the corresponding
// CSV file, and answers domain rank queries against it.
package tranco

import (
	"bufio"
	"bytes"
	"errors"
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

// ErrDomainNotFound is the sentinel wrapped by the error Rank returns when
// the queried domain does not appear in the list. Callers that need to
// distinguish "not found" from an I/O or network failure should check for
// it with errors.Is.
var ErrDomainNotFound = errors.New("domain not found in tranco list")

// TrancoList represents one dated Tranco List (e.g. "the full list for
// 2024-10-01"). Create one with NewTrancoList, then look up ranks with
// Rank. A TrancoList is safe for concurrent use.
type TrancoList struct {
	// ID is the Tranco-assigned list ID resolved from Date/IncludeSubdomain
	// by NewTrancoList (e.g. "XJNYN").
	ID string
	// Date is the list's date in "2006-01-02" format.
	Date string
	// IncludeSubdomain selects the full-domain (fqdn) list when true, or the
	// second-level-domain (sld) list when false.
	IncludeSubdomain bool
	// Scale is the requested list size, e.g. "1000" or "full".
	Scale string
	// CacheFolder is where the downloaded list CSV is cached. A relative
	// path is resolved under the user's home directory; an absolute path is
	// used as-is. Defaults to ".tranco" when empty.
	CacheFolder string
	cache       map[string]int64
	loaded      bool
	cacheMu     sync.Mutex
	httpClient  *http.Client
	userAgent   string
	baseURL     string
	quiet       bool
}

func (t *TrancoList) resolvedBaseURL() string {
	if t.baseURL != "" {
		return t.baseURL
	}
	return defaultBaseURL
}

// Option configures optional TrancoList behavior. Pass one or more to
// NewTrancoList.
type Option func(*TrancoList)

// WithQuiet disables the download progress bar, which otherwise
// unconditionally writes to stderr. Useful for library or server callers
// that don't want that output.
func WithQuiet() Option {
	return func(t *TrancoList) {
		t.quiet = true
	}
}

// NewTrancoList resolves the Tranco list ID for the given date and
// downloads it (or reuses an already-downloaded copy from cacheFolder). See
// TrancoList's field docs for what date, includeSubdomain, scale, and
// cacheFolder mean.
func NewTrancoList(date string, includeSubdomain bool, scale string, cacheFolder string, opts ...Option) (*TrancoList, error) {
	slog.Debug("obtaining tranco list id", slog.String("date", date), slog.Bool("includeSubdomain", includeSubdomain), slog.String("scale", scale))
	list := TrancoList{
		Date:             date,
		IncludeSubdomain: includeSubdomain,
		Scale:            scale,
		httpClient:       &http.Client{Timeout: defaultHTTPTimeout},
		userAgent:        fmt.Sprintf("%s Go-http-client/1.1 tranco-go/%s", strings.Replace(runtime.Version(), "go", "go/", 1), version.PV.Version),
		CacheFolder:      cacheFolder,
	}
	for _, opt := range opts {
		opt(&list)
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

// URL returns the download URL for this list's CSV file.
func (t *TrancoList) URL() string {
	return fmt.Sprintf("%s/download/%s/%s", t.resolvedBaseURL(), t.ID, t.Scale)
}

// Rank returns domain's rank in the list, reading and caching entries from
// the cached CSV file as needed. It returns an error if domain isn't found
// in the list, or if the list file can't be read.
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
		return 0, fmt.Errorf("%w: %s", ErrDomainNotFound, domain)
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

	return 0, fmt.Errorf("%w: %s", ErrDomainNotFound, domain)
}

// DefaultFilePath returns the local path where this list's CSV file is (or
// will be) cached, creating CacheFolder if necessary.
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

// Download fetches this list's CSV file to filePath, unless a file already
// exists there. The download is written atomically: a partial or failed
// download never leaves a file at filePath that a later call would mistake
// for a completed one.
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

	var bar *progressbar.ProgressBar
	if t.quiet {
		bar = progressbar.DefaultBytesSilent(response.ContentLength, "downloading")
	} else {
		bar = progressbar.DefaultBytes(response.ContentLength, "downloading")
	}

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

// Version returns this program's version tag, as recorded at build time.
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
