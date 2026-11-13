package bench

import (
	"context"
	"github.com/Ishitajoshii/stackguard/lab/mcp"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRealWorkloadMeasuresConcurrency(t *testing.T) {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "file"), []byte("actual file content"), 0600)
	work := mcp.NewWorkload(d, 8)
	server := httptest.NewServer(mcp.Server{Tools: []mcp.Tool{{Name: "repository_file"}}, Call: work.Run}.HTTP())
	defer server.Close()
	report, e := Load(context.Background(), LoadOptions{URL: server.URL, Path: "file", Output: filepath.Join(d, "results"), Concurrency: 8, Requests: 16})
	if e != nil || report["server_peak_inflight"].(int64) < 8 {
		t.Fatal(report, e)
	}
}
