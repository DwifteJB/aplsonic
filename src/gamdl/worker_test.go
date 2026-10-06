package gamdl

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

const stubWorker = `
import json, sys, time, threading
out = sys.stdout
print(json.dumps({"event": "ready"}), file=out, flush=True)
lock = threading.Lock()
def handle(req):
    time.sleep(0.05 if req["id"] != "1" else 0.2)
    with lock:
        print(json.dumps({"id": req["id"], "ok": req["url"] != "bad", "paths": [req["output_path"] + "/x.m4a"], "error": "boom"}), file=out, flush=True)
for line in sys.stdin:
    threading.Thread(target=handle, args=(json.loads(line),)).start()
`

func TestWorkerRoundTrip(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available")
	}
	script := filepath.Join(t.TempDir(), "stub.py")
	if err := os.WriteFile(script, []byte(stubWorker), 0o600); err != nil {
		t.Fatal(err)
	}
	w, err := startWorkerWith([]string{python}, script, "tester:abc", "cookies", "aac-web")
	if err != nil {
		t.Fatal(err)
	}
	defer w.stop()

	var wg sync.WaitGroup
	results := make([]error, 4)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			url := "ok"
			if i == 2 {
				url = "bad"
			}
			paths, err := w.Download(context.Background(), url, "/out")
			results[i] = err
			if err == nil && (len(paths) != 1 || paths[0] != "/out/x.m4a") {
				t.Errorf("unexpected paths %v", paths)
			}
		}(i)
	}
	wg.Wait()
	for i, err := range results {
		if i == 2 && (err == nil || err.Error() != "boom") {
			t.Errorf("expected boom error, got %v", err)
		}
		if i != 2 && err != nil {
			t.Errorf("request %d failed: %v", i, err)
		}
	}

	w.stop()
	<-w.done
	if _, err := w.Download(context.Background(), "ok", "/out"); err != ErrWorkerUnavailable {
		t.Errorf("expected unavailable after exit, got %v", err)
	}
}
