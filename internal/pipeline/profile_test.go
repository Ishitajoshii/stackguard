package pipeline

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func checkProfile(t *testing.T, path string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if n, err := io.Copy(io.Discard, r); err != nil || n == 0 {
		t.Fatalf("incomplete profile %s: %d bytes, %v", path, n, err)
	}
}

func TestProfileRotationAndCancellation(t *testing.T) {
	dir := t.TempDir()
	cpu, heap := filepath.Join(dir, "cpu.pprof"), filepath.Join(dir, "heap.pprof")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finish, err := profiles(ctx, cpu, heap, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer finish()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(cpu + ".part-000001"); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("CPU profile did not rotate")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	if err := finish(); err != nil {
		t.Fatal(err)
	}
	if err := finish(); err != nil {
		t.Fatal("finish must be idempotent", err)
	}
	if _, err := os.Stat(heap); !os.IsNotExist(err) {
		t.Fatal("cancelled run must skip heap collection")
	}
	f, err := os.Open(cpu + ".segments.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		var segment profileSegment
		if err := json.Unmarshal(scanner.Bytes(), &segment); err != nil {
			t.Fatal(err)
		}
		if segment.Index != count || segment.Seconds <= 0 || filepath.Base(segment.File) != segment.File {
			t.Fatalf("invalid segment metadata: %+v", segment)
		}
		checkProfile(t, filepath.Join(dir, segment.File))
		count++
	}
	if scanner.Err() != nil || count < 2 {
		t.Fatal("missing finalized segments", count, scanner.Err())
	}
}

func TestProfileRotationRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	cpu := filepath.Join(dir, "cpu.pprof")
	blocked := cpu + ".part-000001"
	if err := os.WriteFile(blocked, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	session, err := startCPUProfiles(context.Background(), cpu, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer session.finish()
	select {
	case <-session.done:
	case <-time.After(5 * time.Second):
		t.Fatal("profile rotation did not report the creation failure")
	}
	if err := session.finish(); err == nil {
		t.Fatal("rotation creation failure was lost")
	}
	b, err := os.ReadFile(blocked)
	if err != nil || string(b) != "preserve" {
		t.Fatal("existing profile overwritten", err)
	}
	checkProfile(t, cpu)
}

func TestCompletedProfilesAndHeap(t *testing.T) {
	dir := t.TempDir()
	cpu, heap := filepath.Join(dir, "cpu.pprof"), filepath.Join(dir, "heap.pprof")
	finish, err := Profiles(context.Background(), cpu, heap)
	if err != nil {
		t.Fatal(err)
	}
	if err := finish(); err != nil {
		t.Fatal(err)
	}
	checkProfile(t, cpu)
	checkProfile(t, heap)
}
