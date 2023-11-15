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
	"strconv"
	"strings"

	"github.com/schollz/progressbar/v3"
)

type TrancoList struct {
	ID               string
	Date             string
	IncludeSubdomain bool
	Scale            string
	cache            map[string]int64
}

func NewTrancoList(date string, includeSubdomain bool, scale string) (*TrancoList, error) {
	slog.Debug("obtaining tranco list id", slog.String("date", date), slog.Bool("includeSubdomain", includeSubdomain), slog.String("scale", scale))
	listID, err := getTrancoListID(date, includeSubdomain)
	if err != nil {
		return nil, err
	}
	slog.Debug("downloading tranco list", slog.String("id", listID))
	list := TrancoList{
		ID:               listID,
		Date:             date,
		IncludeSubdomain: includeSubdomain,
		Scale:            scale,
	}
	list.Download(list.DefaultFilePath())
	slog.Debug("tranco list downloaded", slog.String("id", listID))
	return &list, nil
}

func (t *TrancoList) URL() string {
	return "https://tranco-list.eu/download/" + t.ID + "/" + t.Scale
}

func (t *TrancoList) Rank(domain string) (int64, error) {
	// load from cache
	if t.cache == nil {
		t.cache = make(map[string]int64)
	}

	if rank, ok := t.cache[domain]; ok {
		return rank, nil
	}

	fd, err := os.Open(t.DefaultFilePath())
	if err != nil {
		return 0, err
	}
	defer fd.Close()

	scanner := bufio.NewScanner(fd)
	slog.Debug("Scanning tranco list", slog.String("domain", t.DefaultFilePath()))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		currentRank, currentDomain := parseLine(line)
		slog.Debug("Scanning tranco list", slog.String("domain", currentDomain), slog.Int64("rank", currentRank))
		t.cache[currentDomain] = currentRank
		if currentDomain == domain {
			return currentRank, nil
		}
	}

	return 0, fmt.Errorf("domain %s not found in tranco list", domain)
}

func (t *TrancoList) Download(filePath string) error {
	if _, err := os.Stat(filePath); err == nil {
		return nil
	}

	url := t.URL()

	slog.Info("downloading", slog.String("from", url), slog.String("to", filePath))

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fd, err := os.CreateTemp("", "")
	if err != nil {
		return err
	}
	defer fd.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"downloading",
	)
	_, err = io.Copy(io.MultiWriter(fd, bar), resp.Body)
	if err != nil {
		return err
	}

	err = os.Rename(fd.Name(), filePath)
	if err != nil {
		return err
	}

	slog.Info("downloaded", slog.String("filepath", filePath))
	return nil
}

func (t *TrancoList) DefaultFilePath() string {
	var listType string
	if t.IncludeSubdomain {
		listType = "fqdn"
	} else {
		listType = "sld"
	}
	var baseFolder string
	baseFolder, err := os.UserHomeDir()
	if err != nil {
		baseFolder = os.TempDir()
	}
	folder := filepath.Join(
		baseFolder,
		".tranco",
	)
	filename := fmt.Sprintf("%s_%s_%s_%s.csv", t.Date, listType, t.Scale, t.ID)
	err = os.MkdirAll(folder, 0755)
	if err != nil {
		panic(err)
	}
	return filepath.Join(folder, filename)
}

func getTrancoListID(date string, subdomain bool) (string, error) {
	urlObject := url.URL{
		Scheme: "https",
		Host:   "tranco-list.eu",
		Path:   "daily_list_id",
	}
	query := urlObject.Query()
	query.Set("date", date)
	query.Set("subdomains", strconv.FormatBool(subdomain))
	urlObject.RawQuery = query.Encode()

	response, err := http.Get(urlObject.String())
	if err != nil {
		slog.Error("error occurs when sending HTTP request", slog.String("url", urlObject.String()), slog.String("error", err.Error()))
		return "", err
	}

	if response.StatusCode != 200 {
		slog.Error("error occurs when sending HTTP request", slog.String("url", urlObject.String()), slog.Int("statusCode", response.StatusCode))
		return "", fmt.Errorf("HTTP status code %d", response.StatusCode)
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		slog.Error("error occurs when reading HTTP response body", slog.String("url", urlObject.String()), slog.String("error", err.Error()))
		return "", err
	}

	if bytes.Equal(body, []byte("null")) {
		slog.Error("no list id for %s, api returns null", slog.String("date", date))
		return "", fmt.Errorf("no list id for %s, api returns null", date)
	}

	if bytes.Equal(body, []byte("500 Internal Server Error")) {
		slog.Error("no list id for %s, api returns 500 Internal Server Error", slog.String("date", date))
		return "", fmt.Errorf("no list id for %s, api returns 500 Internal Server Error", date)
	}

	return string(body), nil
}

func parseLine(line string) (int64, string) {
	var rank int64 = 0
	var domain string = ""

	parts := strings.Split(line, ",")

	if len(parts) != 2 {
		return rank, domain
	}

	domain = parts[1]

	rankStr := parts[0]
	rank, err := strconv.ParseInt(rankStr, 10, 64)

	if err != nil {
		return rank, domain
	}

	return rank, domain
}
