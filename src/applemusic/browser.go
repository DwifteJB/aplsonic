package applemusic

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

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
	l := launcher.New().Headless(true)
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
	js := fmt.Sprintf(`async () => {
		const m = MusicKit.getInstance();
		try {
			const r = await m.api.music(%s, %s);
			return JSON.stringify(r.data);
		} catch (e) {
			return "__ERR__" + (e && e.message ? e.message : String(e));
		}
	}`, pathJSON, paramsJSON)

	obj, err := bPage.Eval(js)
	if err != nil {
		//. browser crashed?
		resetBrowser()
		if err2 := ensureBrowser(); err2 != nil {
			return nil, err2
		}
		obj, err = bPage.Eval(js)
		if err != nil {
			return nil, err
		}
	}

	body := obj.Value.Str()
	if strings.HasPrefix(body, "__ERR__") {
		return nil, fmt.Errorf("apple music api (browser): %s", strings.TrimPrefix(body, "__ERR__"))
	}
	return []byte(body), nil
}
