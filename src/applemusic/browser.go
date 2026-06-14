package applemusic

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// NewLauncher builds a go-rod launcher, pointing it at the ROD_BROWSER_BIN
// binary when set (e.g. the chromium baked into the docker image) instead of
// letting go-rod try to download one.
func NewLauncher() *launcher.Launcher {
	l := launcher.New()
	if bin := os.Getenv("ROD_BROWSER_BIN"); bin != "" {
		l = l.Bin(bin)
	}
	return l
}

// this allow us to use actual apple apis! http clients are blocked now :(
const browseURL = "https://music.apple.com/us/browse"

var (
	browserMu sync.Mutex
	bLauncher *launcher.Launcher
	bBrowser  *rod.Browser
	bPage     *rod.Page
)

// keep alive
func Warm() {
	browserMu.Lock()
	defer browserMu.Unlock()
	if err := ensureBrowser(); err != nil {
		fmt.Printf("applemusic: browser warm-up failed (will retry on first request): %v\n", err)
		return
	}
	fmt.Println("applemusic: browser ready")
}

func ensureBrowser() error {
	if bPage != nil {
		return nil
	}
	l := NewLauncher().Headless(true)
	controlURL, err := l.Launch()
	if err != nil {
		return fmt.Errorf("launch browser: %w", err)
	}
	b := rod.New().ControlURL(controlURL)
	if err := b.Connect(); err != nil {
		l.Cleanup()
		return fmt.Errorf("connect browser: %w", err)
	}
	page, err := b.Page(proto.TargetCreateTarget{URL: browseURL})
	if err != nil {
		b.Close()
		l.Cleanup()
		return fmt.Errorf("open page: %w", err)
	}
	_ = page.WaitLoad()

	deadline := time.Now().Add(40 * time.Second)
	for {
		obj, err := page.Eval(`() => { try { return !!(window.MusicKit && MusicKit.getInstance()); } catch(e) { return false; } }`)
		if err == nil && obj.Value.Bool() {
			break
		}
		if time.Now().After(deadline) {
			b.Close()
			l.Cleanup()
			return fmt.Errorf("MusicKit did not initialise in the browser")
		}
		time.Sleep(500 * time.Millisecond)
	}

	bLauncher = l
	bBrowser = b
	bPage = page
	return nil
}

func resetBrowser() {
	if bBrowser != nil {
		bBrowser.Close()
	}
	if bLauncher != nil {
		bLauncher.Cleanup()
	}
	bLauncher = nil
	bBrowser = nil
	bPage = nil
}

// basically fetching
func browserGet(path string, query url.Values) ([]byte, error) {
	return browserDo("GET", path, query, nil, "")
}

func browserDo(method, path string, query url.Values, body any, mediaUserToken string) ([]byte, error) {
	browserMu.Lock()
	defer browserMu.Unlock()

	if err := ensureBrowser(); err != nil {
		return nil, err
	}

	params := map[string]string{}
	for k := range query {
		params[k] = query.Get(k)
	}
	pathJSON, _ := json.Marshal(path)
	paramsJSON, _ := json.Marshal(params)

	bodyJSON := []byte("null")
	if body != nil {
		if b, err := json.Marshal(body); err == nil {
			bodyJSON = b
		}
	}
	methodJSON, _ := json.Marshal(strings.ToUpper(method))
	mutJSON, _ := json.Marshal(mediaUserToken)

	js := fmt.Sprintf(`async () => {
		const m = MusicKit.getInstance();
		const method = %s;
		const path = %s;
		const params = %s;
		const body = %s;
		const mut = %s;
		const devToken = m.developerToken || (m.api && m.api.developerToken) || "";
		try {
			if (!mut && method === "GET") {
				const r = await m.api.music(path, params);
				return JSON.stringify(r.data ?? {});
			}
			const base = "https://amp-api.music.apple.com";
			let url = base + path;
			const qs = new URLSearchParams(params).toString();
			if (qs) url += (path.indexOf("?") >= 0 ? "&" : "?") + qs;
			const headers = {
				"Authorization": "Bearer " + devToken,
				"Origin": "https://music.apple.com",
			};
			if (mut) headers["Music-User-Token"] = mut;
			if (body !== null) headers["Content-Type"] = "application/json";
			const res = await fetch(url, {
				method,
				headers,
				body: body !== null ? JSON.stringify(body) : undefined,
			});
			const txt = await res.text();
			if (!res.ok && res.status !== 201 && res.status !== 204) {
				return "__ERR__" + res.status + " " + txt;
			}
			return txt && txt.length ? txt : "{}";
		} catch (e) {
			return "__ERR__" + (e && e.message ? e.message : String(e));
		}
	}`, methodJSON, pathJSON, paramsJSON, bodyJSON, mutJSON)

	obj, err := bPage.Eval(js)
	if err != nil {
		resetBrowser()
		if err2 := ensureBrowser(); err2 != nil {
			return nil, err2
		}
		obj, err = bPage.Eval(js)
		if err != nil {
			return nil, err
		}
	}

	out := obj.Value.Str()
	if strings.HasPrefix(out, "__ERR__") {
		return nil, fmt.Errorf("apple music api (browser): %s", strings.TrimPrefix(out, "__ERR__"))
	}
	return []byte(out), nil
}
