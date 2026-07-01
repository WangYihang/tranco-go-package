package tranco_test

import (
	"testing"

	"github.com/WangYihang/tranco-go-package"
	"github.com/WangYihang/tranco-go-package/pkg/version"
)

// TestNewTrancoListIntegration exercises the real tranco-list.eu API and
// download endpoint end-to-end. It's skipped in -short mode (as run in CI)
// since it depends on external network access; run `go test .` without
// -short to exercise it manually.
func TestNewTrancoListIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test that hits the real tranco-list.eu API in short mode")
	}
	list, err := tranco.NewTrancoList("2019-04-30", false, "1000", ".tranco")
	if err != nil {
		t.Fatalf("NewTrancoList() unexpected error: %v", err)
	}
	rank, err := list.Rank("google.com")
	if err != nil {
		t.Fatalf("Rank() unexpected error: %v", err)
	}
	if rank != 1 {
		t.Errorf("Rank(%q) = %d, want %d", "google.com", rank, 1)
	}
}

func BenchmarkDomainLookup(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping integration benchmark that hits the real tranco-list.eu API in short mode")
	}
	list, err := tranco.NewTrancoList("2019-04-30", false, "1000", ".tranco")
	if err != nil {
		b.Fatalf("NewTrancoList() unexpected error: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list.Rank("google.com")
	}
}

func TestVersion(t *testing.T) {
	expectedVersion, err := version.GetVersionFromGit()
	if err != nil {
		t.Error(err)
	}
	if tranco.Version() != expectedVersion {
		t.Error("version mismatch")
	}
}
