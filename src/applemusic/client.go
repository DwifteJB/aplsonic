package applemusic

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	ampAPIURL   = "https://amp-api.music.apple.com"
	apiLanguage = "en-US"
)

type Client struct {
	Storefront string
}

// NewClientFromCookies builds a client from a user's Netscape cookie jar. The
// media-user-token presence is required as a sign the account is set up; the
// storefront comes from the itua cookie (defaulting to "us").
func NewClientFromCookies(netscape string) (*Client, error) {
	mut, storefront := parseCookies(netscape)
	if mut == "" {
		return nil, fmt.Errorf("media-user-token not found in cookies")
	}
	if storefront == "" {
		storefront = "us"
	}
	return &Client{Storefront: storefront}, nil
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

// handles parsing properly 
func MediaTokenExpiry(netscape string) (expiry time.Time, ok bool) {
	for _, line := range strings.Split(netscape, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 7)
		if len(parts) < 7 || parts[5] != "media-user-token" {
			continue
		}
		ts, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil || ts <= 0 {
			return time.Time{}, false
		}
		return time.Unix(ts, 0), true
	}
	return time.Time{}, false
}

// see if the user has proper cookies
func HasMediaUserToken(netscape string) bool {
	mut, _ := parseCookies(netscape)
	return mut != ""
}

func (c *Client) get(path string, params url.Values) ([]byte, error) {
	params.Set("l", apiLanguage)
	return browserGet(path, params)
}

func (c *Client) getURL(fullURL string) ([]byte, error) {
	u, err := url.Parse(fullURL)
	if err != nil {
		return nil, err
	}
	return browserGet(u.Path, u.Query())
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

	// some non-us storefronts return a truncated tracklist; fall back to us if it has more
	if c.Storefront != "us" {
		usTracks, err := c.fetchTracks(albumID, "us")
		if err == nil && len(usTracks) > len(tracks) {
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

	return &wrapper.Results, nil
}
