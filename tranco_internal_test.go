package tranco

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantRank   int64
		wantDomain string
		wantErr    bool
	}{
		{name: "valid", line: "1,google.com", wantRank: 1, wantDomain: "google.com"},
		{name: "valid large rank", line: "1000000,example.org", wantRank: 1000000, wantDomain: "example.org"},
		{name: "empty line", line: "", wantErr: true},
		{name: "missing comma", line: "1 google.com", wantErr: true},
		{name: "too many fields", line: "1,google.com,extra", wantErr: true},
		{name: "non-numeric rank", line: "abc,google.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rank, domain, err := parseLine(tt.line)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseLine(%q) expected an error, got none", tt.line)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseLine(%q) unexpected error: %v", tt.line, err)
			}
			if rank != tt.wantRank || domain != tt.wantDomain {
				t.Errorf("parseLine(%q) = (%d, %q), want (%d, %q)", tt.line, rank, domain, tt.wantRank, tt.wantDomain)
			}
		})
	}
}

func newTestTrancoList(t *testing.T, baseURL string) *TrancoList {
	t.Helper()
	return &TrancoList{
		Date:             "2024-01-01",
		IncludeSubdomain: false,
		Scale:            "1000",
		CacheFolder:      t.TempDir(),
		httpClient:       http.DefaultClient,
		userAgent:        "tranco-go-test",
		baseURL:          baseURL,
	}
}

func TestGetTrancoListID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/daily_list_id" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, "ABCDE")
	}))
	defer server.Close()

	list := newTestTrancoList(t, server.URL)
	id, err := list.getTrancoListID("2024-01-01", false)
	if err != nil {
		t.Fatalf("getTrancoListID() unexpected error: %v", err)
	}
	if id != "ABCDE" {
		t.Errorf("getTrancoListID() = %q, want %q", id, "ABCDE")
	}
}

func TestGetTrancoListIDNullResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "null")
	}))
	defer server.Close()

	list := newTestTrancoList(t, server.URL)
	if _, err := list.getTrancoListID("2024-01-01", false); err == nil {
		t.Fatal("getTrancoListID() expected an error for a null response, got nil")
	}
}

func TestGetTrancoListIDServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	list := newTestTrancoList(t, server.URL)
	if _, err := list.getTrancoListID("2024-01-01", false); err == nil {
		t.Fatal("getTrancoListID() expected an error for HTTP 500, got nil")
	}
}

func TestDownloadNonOKStatusLeavesNoFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found")
	}))
	defer server.Close()

	list := newTestTrancoList(t, server.URL)
	list.ID = "XXXXX"

	filePath := filepath.Join(t.TempDir(), "list.csv")
	if err := list.Download(filePath); err == nil {
		t.Fatal("Download() expected an error for HTTP 404, got nil")
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("Download() should not leave a file at %s after a failed download", filePath)
	}
	if _, err := os.Stat(filePath + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("Download() should not leave a temp file at %s.tmp after a failed download", filePath)
	}
}

func TestDownloadSuccess(t *testing.T) {
	const body = "1,google.com\n2,example.com\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer server.Close()

	list := newTestTrancoList(t, server.URL)
	list.ID = "XXXXX"

	filePath := filepath.Join(t.TempDir(), "list.csv")
	if err := list.Download(filePath); err != nil {
		t.Fatalf("Download() unexpected error: %v", err)
	}

	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(got) != body {
		t.Errorf("downloaded content = %q, want %q", got, body)
	}
}

func TestRank(t *testing.T) {
	list := &TrancoList{
		Date:             "2024-01-01",
		IncludeSubdomain: false,
		Scale:            "1000",
		CacheFolder:      t.TempDir(),
		ID:               "TESTID",
	}

	filePath, err := list.DefaultFilePath()
	if err != nil {
		t.Fatalf("DefaultFilePath() unexpected error: %v", err)
	}

	content := "1,google.com\n2,example.com\nmalformed-line\n4,tail.com\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test list file: %v", err)
	}

	rank, err := list.Rank("example.com")
	if err != nil {
		t.Fatalf("Rank() unexpected error: %v", err)
	}
	if rank != 2 {
		t.Errorf("Rank(%q) = %d, want %d", "example.com", rank, 2)
	}

	_, err = list.Rank("missing.com")
	if err == nil {
		t.Fatal("Rank() expected an error for a domain not in the list")
	}
	if !errors.Is(err, ErrDomainNotFound) {
		t.Errorf("Rank() error = %v, want it to wrap ErrDomainNotFound", err)
	}
	if !list.loaded {
		t.Error("Rank() should mark the list as fully loaded after scanning through EOF")
	}

	// A second lookup for a different absent domain should short-circuit via
	// the loaded cache rather than re-scanning, and still wrap the sentinel.
	_, err = list.Rank("still-missing.com")
	if !errors.Is(err, ErrDomainNotFound) {
		t.Errorf("Rank() error = %v, want it to wrap ErrDomainNotFound", err)
	}
}
