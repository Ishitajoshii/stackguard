package bench

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/output"
	"github.com/Ishitajoshii/stackguard/internal/pipeline"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func Replay(ctx context.Context, input, sourceManifest, destination string, count int64) error {
	return ReplayWithSessions(ctx, input, sourceManifest, destination, count, "copies")
}

// ReplayWithSessions uses copies for independent sessions or single for one
// deliberately merged load-test session. Neither mode creates agent evidence.
func ReplayWithSessions(ctx context.Context, input, sourceManifest, destination string, count int64, sessions string) error {
	if sessions != "copies" && sessions != "single" {
		return errors.New("replay sessions must be copies or single")
	}
	if count < 1 || count > 10000000 {
		return errors.New("replay count must be 1..10000000")
	}
	source, e := output.ChecksumContext(ctx, input)
	if e != nil {
		return e
	}
	manifestSource, e := output.ChecksumContext(ctx, sourceManifest)
	if e != nil {
		return errors.New("original capture manifest required")
	}
	f, e := os.Open(sourceManifest)
	if e != nil {
		return e
	}
	var original model.Manifest
	e = json.NewDecoder(f).Decode(&original)
	f.Close()
	if e != nil || original.Version != 1 || original.RunID == "" {
		return errors.New("invalid original capture manifest")
	}
	w, e := output.New(filepath.Join(destination, "replay.jsonl"))
	if e != nil {
		return e
	}
	defer w.Close()
	manifest := output.Manifest("throughput_replay")
	manifest.Sources = []model.Source{source, manifestSource}
	manifest.Notes = []string{"THROUGHPUT ONLY: derived copies are not new agent behavior or correctness evidence.", "All event/session/span/request identifiers and their parent references are consistently rewritten per copy. NHI identities are preserved.", "Original capture completeness: " + strconv.FormatBool(original.Complete)}
	if sessions == "single" {
		manifest.Notes[1] = "Single-session load shape: source sessions are deliberately merged; span/request keys retain their original session namespace and copy number. Parent references use the same mapping. NHI identities are preserved."
	}
	manifest.Notes = append(manifest.Notes, "Session mode: "+sessions)
	var records int64
	var runErr error
	for copy := int64(0); records < count; copy++ {
		if ctx.Err() != nil {
			runErr = ctx.Err()
			break
		}
		f, e := os.Open(input)
		if e != nil {
			runErr = e
			break
		}
		before := records
		prefix := strconv.FormatInt(copy, 10) + ":"
		rewrite := func(s string) string {
			if s == "" {
				return ""
			}
			return secrets.Fingerprint(prefix + s)
		}
		e = pipeline.Lines(ctx, f, func(line string) error {
			if records >= count {
				return nil
			}
			var v model.Event
			if json.Unmarshal([]byte(line), &v) != nil || v.Validate() != nil {
				return errors.New("replay requires validated normalized original events")
			}
			v.ID = rewrite(v.ID)
			if sessions == "single" {
				own, parent := v.Session, v.ParentSession
				if parent == "" {
					parent = own
				}
				namespaced := func(session, id string) string {
					if id == "" {
						return ""
					}
					return rewrite(session + "\x00" + id)
				}
				v.Span = namespaced(own, v.Span)
				v.Parent = namespaced(parent, v.Parent)
				v.CallID = namespaced(own, v.CallID)
				v.RequestID = namespaced(own, v.RequestID)
				v.ParentRequestID = namespaced(parent, v.ParentRequestID)
				v.Session = secrets.Fingerprint("single-session:" + source.SHA256)
				if v.ParentSession != "" {
					v.ParentSession = v.Session
				}
			} else {
				v.Session = rewrite(v.Session)
				v.Span = rewrite(v.Span)
				v.Parent = rewrite(v.Parent)
				v.ParentSession = rewrite(v.ParentSession)
				v.CallID = rewrite(v.CallID)
				v.RequestID = rewrite(v.RequestID)
				v.ParentRequestID = rewrite(v.ParentRequestID)
			}
			if e = w.Write(v); e != nil {
				return e
			}
			records++
			return nil
		})
		f.Close()
		if e != nil {
			runErr = e
			break
		}
		if records == before {
			runErr = errors.New("original capture contains no events")
			break
		}
	}
	runErr = errors.Join(runErr, w.Close())
	manifest.Records = uint64(records)
	manifest.Ended = time.Now().UTC()
	manifest.Complete = runErr == nil
	if runErr != nil {
		manifest.Errors = 1
	}
	return errors.Join(runErr, output.Publish(filepath.Join(destination, "manifest.json"), manifest))
}
