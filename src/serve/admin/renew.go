package admin

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/DwifteJB/aplsonic/src/applemusic"
	"github.com/DwifteJB/aplsonic/src/config"
	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)


const renewURL = "https://music.apple.com"

func attemptRenew(u *schema.User) (ok bool) {
	if !config.AppConfig.TokenAutoRenew {
		return false
	}
	if !applemusic.HasCookie(u.AppleCookies, "myacinfo") {
		fmt.Printf("token monitor: %q has no myacinfo cookie, cannot silently renew (re-upload a full cookies.txt)\n", u.Username)
		return false
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("token monitor: renew for %q panicked: %v\n", u.Username, r)
			ok = false
		}
	}()

	jar, err := silentRenew(u.AppleCookies)
	if err != nil {
		fmt.Printf("token monitor: renew for %q failed: %v\n", u.Username, err)
		return false
	}

	u.AppleCookies = jar
	now := time.Now()
	updates := map[string]interface{}{
		"apple_cookies":               jar,
		"apple_token_last_checked_at": now,
	}
	if exp, found := applemusic.MediaTokenExpiry(jar); found {
		updates["apple_token_expires_at"] = exp
		u.AppleTokenExpiresAt = &exp
	}
	db.DB.Model(u).Updates(updates)
	return true
}

// silentRenew drives a headless browser through MusicKit authorize() using the
// supplied cookie jar, and returns a freshly re-captured Netscape jar.
func silentRenew(netscape string) (string, error) {
	l := launcher.New().Headless(true)
	controlURL, err := l.Launch()
	if err != nil {
		return "", fmt.Errorf("launch browser: %w", err)
	}
	defer l.Cleanup()
	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return "", fmt.Errorf("connect browser: %w", err)
	}
	defer browser.Close()

	// inject the stored cookies before navigating
	var params []*proto.NetworkCookieParam
	for _, c := range applemusic.ParseNetscape(netscape) {
		p := &proto.NetworkCookieParam{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
			Path:   c.Path,
			Secure: c.Secure,
		}
		if c.Expires > 0 {
			p.Expires = proto.TimeSinceEpoch(float64(c.Expires))
		}
		params = append(params, p)
	}
	if err := browser.SetCookies(params); err != nil {
		return "", fmt.Errorf("set cookies: %w", err)
	}

	page, err := browser.Page(proto.TargetCreateTarget{URL: renewURL})
	if err != nil {
		return "", fmt.Errorf("open page: %w", err)
	}
	_ = page.WaitLoad()

	const js = `async () => {
		if (!window.MusicKit) return JSON.stringify({state:"loading"});
		let m;
		try { m = MusicKit.getInstance(); } catch(e) { return JSON.stringify({state:"loading"}); }
		if (!m) return JSON.stringify({state:"loading"});
		if (!m.isAuthorized) return JSON.stringify({state:"unauthorized"});
		try {
			const t = await m.authorize();
			return JSON.stringify({state:"ok", token: t || "", storefront: m.storefrontId || ""});
		} catch(e) { return JSON.stringify({state:"error", error: String(e)}); }
	}`

	var result struct {
		State, Token, Storefront, Error string
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		obj, err := page.Eval(js)
		if err == nil {
			_ = json.Unmarshal([]byte(obj.Value.Str()), &result)
			if result.State != "" && result.State != "loading" {
				break
			}
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("timed out waiting for MusicKit (last state %q)", result.State)
		}
		time.Sleep(500 * time.Millisecond)
	}

	switch result.State {
	case "unauthorized":
		return "", fmt.Errorf("session not authorized (auth cookies expired)")
	case "error":
		return "", fmt.Errorf("authorize() failed: %s", result.Error)
	case "ok":
		if result.Token == "" {
			return "", fmt.Errorf("authorize() returned an empty token")
		}
	default:
		return "", fmt.Errorf("unexpected state %q", result.State)
	}

	// re-capture the full (refreshed) jar from the browser
	raw, err := page.Cookies([]string{})
	if err != nil {
		return "", fmt.Errorf("read cookies: %w", err)
	}
	cookies := make([]applemusic.Cookie, 0, len(raw))
	for _, c := range raw {
		cookies = append(cookies, applemusic.Cookie{
			Domain:  c.Domain,
			Path:    c.Path,
			Secure:  c.Secure,
			Expires: int64(c.Expires),
			Name:    c.Name,
			Value:   c.Value,
		})
	}
	jar := applemusic.RenderNetscape(cookies)
	if !applemusic.HasMediaUserToken(jar) {
		return "", fmt.Errorf("refreshed jar has no media-user-token")
	}
	return jar, nil
}
