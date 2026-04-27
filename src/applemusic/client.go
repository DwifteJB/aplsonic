package applemusic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	homepageURL = "https://beta.music.apple.com"
	ampAPIURL   = "https://amp-api.music.apple.com"
	apiLanguage = "en-US"
	
	// since we are emulating a "web" client
	userAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:95.0) Gecko/20100101 Firefox/95.0"
)

var (
	tokenCache  string
	tokenExpiry time.Time
	tokenMu     sync.Mutex
)

// a client for making auth requests to apple music api
type Client struct {
	httpClient     *http.Client
	mediaUserToken string
	Storefront     string
}

// using the cookies from cmd/createAccount for auth
func NewClientFromCookies(netscape string) (*Client, error) {
	mut, storefront := parseCookies(netscape)
	if mut == "" {
		return nil, fmt.Errorf("media-user-token not found in cookies")
	}
	if storefront == "" {
		storefront = "us"
	}
	return &Client{
		httpClient:     &http.Client{Timeout: 15 * time.Second},
		mediaUserToken: mut,
		Storefront:     storefront,
	}, nil
}

func parseCookies(netscape string) (mediaUserToken, storefront string) {
	for _, line := range strings.Split(netscape, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 7)
		if len(parts) < 7 {
			continue
		}
		name, value := parts[5], parts[6]
		switch name {
		case "media-user-token":
			mediaUserToken = value
		case "itua":
			storefront = strings.ToLower(value)
		}
	}
	return
}


// jsURLPatterns are tried in order against the Apple Music homepage HTML.
// sometimes this changes? soooo we have to try find whatever we can
var jsURLPatterns = []*regexp.Regexp{
	regexp.MustCompile(`/assets/index-legacy-[^"'\s>]+\.js`),
	regexp.MustCompile(`/assets/index-[^"'\s>]+\.js`),
	regexp.MustCompile(`/assets/[^"'\s>]+\.js`),
}

var tokenRe = regexp.MustCompile(`eyJh[A-Za-z0-9._-]{20,}`)

// getBearerToken returns the app-level bearer token embedded in the Apple Music
// web app JS bundle, refreshing it when the 1-hour cache expires.
// this is the only way to get a proper token without a headless browesr instance, like ROD
func (c *Client) getBearerToken() (string, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	if tokenCache != "" && time.Now().Before(tokenExpiry) {
		return tokenCache, nil
	}

	req, _ := http.NewRequest("GET", homepageURL, nil)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching Apple Music homepage: %w", err)
	}
	defer resp.Body.Close()
	pageBody, _ := io.ReadAll(resp.Body)

	// trry get token from HTML
	if tm := tokenRe.Find(pageBody); tm != nil {
		tokenCache = string(tm)
		tokenExpiry = time.Now().Add(time.Hour)
		return tokenCache, nil
	}

	// scan JS and try find the token there
	var jsPath string
	for _, re := range jsURLPatterns {
		if m := re.Find(pageBody); m != nil {
			jsPath = string(m)
			break
		}
	}
	if jsPath == "" {
		return "", fmt.Errorf("could not locate Apple Music JS bundle (homepage returned %d bytes)", len(pageBody))
	}

	jsURL := homepageURL + jsPath
	jsResp, err := c.httpClient.Get(jsURL)
	if err != nil {
		return "", fmt.Errorf("fetching Apple Music JS bundle %s: %w", jsURL, err)
	}
	defer jsResp.Body.Close()
	jsBody, _ := io.ReadAll(jsResp.Body)

	tm := tokenRe.Find(jsBody)
	if tm == nil {
		return "", fmt.Errorf("could not extract bearer token from %s", jsURL)
	}

	tokenCache = string(tm)
	tokenExpiry = time.Now().Add(time.Hour)
	return tokenCache, nil
}

func (c *Client) getURL(fullURL string) ([]byte, error) {
	token, err := c.getBearerToken()
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequest("GET", fullURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Media-User-Token", c.mediaUserToken)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", homepageURL)
	req.Header.Set("Referer", homepageURL+"/")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Apple Music API request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Apple Music API returned %d: %s", resp.StatusCode, body)
	}
	return body, nil
}

func (c *Client) get(path string, params url.Values) ([]byte, error) {
	token, err := c.getBearerToken()
	if err != nil {
		return nil, err
	}

	params.Set("l", apiLanguage)
	u := ampAPIURL + path + "?" + params.Encode()

	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Media-User-Token", c.mediaUserToken)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", homepageURL)
	req.Header.Set("Referer", homepageURL+"/")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Apple Music API request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Apple Music API %s returned %d: %s", path, resp.StatusCode, body)
	}
	return body, nil
}

func (c *Client) GetAlbum(id string) (*Resource, error) {
	data, err := c.get(fmt.Sprintf("/v1/catalog/%s/albums/%s", c.Storefront, id), url.Values{})
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Data []Resource `json:"data"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("parsing album response: %w", err)
	}
	if len(wrapper.Data) == 0 {
		return nil, fmt.Errorf("album %s not found", id)
	}
	album := &wrapper.Data[0]

	tracks, err := c.getAllTracks(id)
	if err == nil && len(tracks) > 0 {
		album.Relationships.Tracks = &ResourceList{Data: tracks}
	}

	return album, nil
}

func (c *Client) getAllTracks(albumID string) ([]Resource, error) {
	tracks, err := c.fetchTracks(albumID, c.Storefront)
	if err != nil {
		return nil, err
	}
	fmt.Printf("[tracks] storefront=%s albumID=%s count=%d\n", c.Storefront, albumID, len(tracks))

	if c.Storefront != "us" {
		usTracks, err := c.fetchTracks(albumID, "us")
		if err == nil && len(usTracks) > len(tracks) {
			fmt.Printf("[tracks] us storefront returned more tracks (%d > %d), using us\n", len(usTracks), len(tracks))
			return usTracks, nil
		}
	}
	return tracks, nil
}

func (c *Client) fetchTracks(albumID, storefront string) ([]Resource, error) {
	var all []Resource
	nextURL := fmt.Sprintf("%s/v1/catalog/%s/albums/%s/tracks?limit=300&l=%s", ampAPIURL, storefront, albumID, apiLanguage)

	for nextURL != "" {
		data, err := c.getURL(nextURL)
		if err != nil {
			return all, err
		}
		var page ResourceList
		if err := json.Unmarshal(data, &page); err != nil {
			return all, err
		}
		all = append(all, page.Data...)
		if page.Next == "" {
			break
		}
		nextURL = ampAPIURL + page.Next
	}
	return all, nil
}

func (c *Client) GetArtist(id string) (*Resource, error) {
	params := url.Values{
		"include": {"albums"},
	}
	data, err := c.get(fmt.Sprintf("/v1/catalog/%s/artists/%s", c.Storefront, id), params)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Data []Resource `json:"data"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("parsing artist response: %w", err)
	}
	if len(wrapper.Data) == 0 {
		return nil, fmt.Errorf("artist %s not found", id)
	}
	return &wrapper.Data[0], nil
}

// search for apple music
func (c *Client) Search(term string, limit int) (*SearchResults, error) {
	params := url.Values{
		"term":           {term},
		"types":          {"albums,songs,artists"},
		"limit":          {fmt.Sprintf("%d", limit)},
		"include[songs]": {"albums"},
	}

	data, err := c.get(fmt.Sprintf("/v1/catalog/%s/search", c.Storefront), params)
	if err != nil {
		return nil, err
	}
	

	var wrapper struct {
		Results SearchResults `json:"results"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("parsing Apple Music search response: %w", err)
	}

	fmt.Printf("raw search results: %s\n", data)

	return &wrapper.Results, nil
}
