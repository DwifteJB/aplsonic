package gamdl

import (
	"bufio"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed worker.py
var workerScript []byte

var ErrWorkerUnavailable = errors.New("gamdl worker unavailable")

const (
	workerReadyTimeout = 120 * time.Second
	workerConcurrency  = 3
)

type Worker struct {
	key     string
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	mu      sync.Mutex
	pending map[string]chan workerResult
	seq     uint64
	done    chan struct{}
}

type workerResult struct {
	Paths []string
	Err   error
}

type workerMessage struct {
	Event string   `json:"event"`
	ID    string   `json:"id"`
	OK    bool     `json:"ok"`
	Paths []string `json:"paths"`
	Error string   `json:"error"`
}

var (
	poolMu     sync.Mutex
	pool       = map[string]*Worker{}
	scriptOnce sync.Once
	scriptPath string
	scriptErr  error
)

func WorkerFor(username, cookies, codec string) (*Worker, error) {
	sum := sha256.Sum256([]byte(cookies))
	key := username + ":" + hex.EncodeToString(sum[:8])

	poolMu.Lock()
	defer poolMu.Unlock()

	if w, ok := pool[key]; ok && w.alive() {
		return w, nil
	}
	for k, w := range pool {
		if strings.HasPrefix(k, username+":") {
			w.stop()
			delete(pool, k)
		}
	}

	w, err := startWorker(key, cookies, codec)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWorkerUnavailable, err)
	}
	pool[key] = w
	return w, nil
}

func (w *Worker) Download(ctx context.Context, url, outDir string) ([]string, error) {
	if !w.alive() {
		return nil, ErrWorkerUnavailable
	}
	w.mu.Lock()
	w.seq++
	id := strconv.FormatUint(w.seq, 10)
	ch := make(chan workerResult, 1)
	w.pending[id] = ch
	req, _ := json.Marshal(map[string]string{"id": id, "url": url, "output_path": outDir})
	_, err := w.stdin.Write(append(req, '\n'))
	w.mu.Unlock()
	if err != nil {
		w.forget(id)
		return nil, fmt.Errorf("%w: %v", ErrWorkerUnavailable, err)
	}

	select {
	case res := <-ch:
		return res.Paths, res.Err
	case <-ctx.Done():
		w.forget(id)
		return nil, ctx.Err()
	case <-w.done:
		return nil, ErrWorkerUnavailable
	}
}

func (w *Worker) alive() bool {
	select {
	case <-w.done:
		return false
	default:
		return true
	}
}

func (w *Worker) forget(id string) {
	w.mu.Lock()
	delete(w.pending, id)
	w.mu.Unlock()
}

func (w *Worker) stop() {
	if w.cmd.Process != nil {
		_ = w.stdin.Close()
		_ = w.cmd.Process.Kill()
	}
}

func startWorker(key, cookies, codec string) (*Worker, error) {
	script, err := ensureScript()
	if err != nil {
		return nil, err
	}
	python, err := findPython()
	if err != nil {
		return nil, err
	}
	return startWorkerWith(python, script, key, cookies, codec)
}

func startWorkerWith(python []string, script, key, cookies, codec string) (*Worker, error) {

	sum := sha256.Sum256([]byte(key))
	dir := filepath.Join(os.TempDir(), "aplsonic-gamdl", hex.EncodeToString(sum[:8]))
	tmpDir := filepath.Join(dir, "tmp")
	if err := os.MkdirAll(tmpDir, 0o700); err != nil {
		return nil, err
	}
	cookiePath := filepath.Join(dir, "cookies.txt")
	if err := os.WriteFile(cookiePath, []byte(cookies), 0o600); err != nil {
		return nil, err
	}

	args := append(append([]string{}, python[1:]...), script, cookiePath, codec, tmpDir, strconv.Itoa(workerConcurrency))
	cmd := exec.Command(python[0], args...)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	w := &Worker{
		key:     key,
		cmd:     cmd,
		stdin:   stdin,
		pending: map[string]chan workerResult{},
		done:    make(chan struct{}),
	}

	reader := bufio.NewReaderSize(stdout, 1<<20)
	ready := make(chan error, 1)
	go func() {
		line, err := reader.ReadString('\n')
		if err != nil {
			ready <- fmt.Errorf("worker exited before ready: %v", err)
			return
		}
		var msg workerMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			ready <- fmt.Errorf("bad worker handshake: %q", strings.TrimSpace(line))
			return
		}
		if msg.Event != "ready" {
			ready <- errors.New(msg.Error)
			return
		}
		ready <- nil
	}()

	select {
	case err := <-ready:
		if err != nil {
			w.stop()
			_ = cmd.Wait()
			return nil, err
		}
	case <-time.After(workerReadyTimeout):
		w.stop()
		_ = cmd.Wait()
		return nil, errors.New("worker did not become ready in time")
	}

	fmt.Printf("gamdl worker ready for %s\n", strings.SplitN(key, ":", 2)[0])
	go w.readLoop(reader)
	return w, nil
}

func (w *Worker) readLoop(reader *bufio.Reader) {
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			var msg workerMessage
			if json.Unmarshal([]byte(line), &msg) == nil && msg.ID != "" {
				w.mu.Lock()
				ch, ok := w.pending[msg.ID]
				delete(w.pending, msg.ID)
				w.mu.Unlock()
				if ok {
					res := workerResult{Paths: msg.Paths}
					if !msg.OK {
						res.Err = errors.New(msg.Error)
					}
					ch <- res
				}
			}
		}
		if err != nil {
			break
		}
	}
	_ = w.cmd.Wait()
	close(w.done)
	w.mu.Lock()
	for id, ch := range w.pending {
		delete(w.pending, id)
		ch <- workerResult{Err: ErrWorkerUnavailable}
	}
	w.mu.Unlock()
	poolMu.Lock()
	if pool[w.key] == w {
		delete(pool, w.key)
	}
	poolMu.Unlock()
	fmt.Printf("gamdl worker for %s exited\n", strings.SplitN(w.key, ":", 2)[0])
}

func ensureScript() (string, error) {
	scriptOnce.Do(func() {
		dir := filepath.Join(os.TempDir(), "aplsonic-gamdl")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			scriptErr = err
			return
		}
		scriptPath = filepath.Join(dir, "worker.py")
		scriptErr = os.WriteFile(scriptPath, workerScript, 0o600)
	})
	return scriptPath, scriptErr
}

func findPython() ([]string, error) {
	if out, err := exec.Command("uv", "tool", "dir").Output(); err == nil {
		base := filepath.Join(strings.TrimSpace(string(out)), "gamdl", "bin")
		for _, name := range []string{"python3", "python"} {
			if p := filepath.Join(base, name); isExecutable(p) {
				return []string{p}, nil
			}
		}
	}
	if p, err := exec.LookPath("gamdl"); err == nil {
		if f, err := os.Open(p); err == nil {
			line, _ := bufio.NewReader(f).ReadString('\n')
			f.Close()
			if strings.HasPrefix(line, "#!") {
				fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "#!"))
				if len(fields) > 0 && isExecutable(fields[0]) && !strings.Contains(fields[0], "/env") {
					return []string{fields[0]}, nil
				}
			}
		}
	}
	if _, err := exec.LookPath("uv"); err == nil {
		return []string{"uv", "tool", "run", "--from", "gamdl", "python"}, nil
	}
	return nil, errors.New("no python with gamdl found")
}

func isExecutable(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}
