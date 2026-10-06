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
	Storefront     string
	MediaUserToken string
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
	return &Client{Storefront: storefront, MediaUserToken: mut}, nil
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

func (c *Client) getMe(fullURL string) ([]byte, error) {
	u, err := url.Parse(fullURL)
	if err != nil {
		return nil, err
	}
	return browserDo("GET", u.Path, u.Query(), nil, c.MediaUserToken)
}

// gets the users playlists w/ pagination
func (c *Client) GetLibraryPlaylists() ([]Resource, error) {
	var all []Resource
	nextURL := fmt.Sprintf("%s/v1/me/library/playlists?limit=100&l=%s", ampAPIURL, apiLanguage)
	for nextURL != "" {
		data, err := c.getMe(nextURL)
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

// gets the tracks within the library playlist
func (c *Client) GetLibraryPlaylistTracks(playlistID string) ([]Resource, error) {
	var all []Resource
	nextURL := fmt.Sprintf("%s/v1/me/library/playlists/%s/tracks?limit=100&l=%s", ampAPIURL, playlistID, apiLanguage)
	for nextURL != "" {
		data, err := c.getMe(nextURL)
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

type trackRef struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// converts a list of catalog song ids into the format needed for playlist creation
func catalogTrackData(catalogSongIDs []string) []trackRef {
	data := make([]trackRef, 0, len(catalogSongIDs))
	for _, id := range catalogSongIDs {
		if id == "" {
			continue
		}
		data = append(data, trackRef{ID: id, Type: "songs"})
	}
	return data
}


// create a playlist
func (c *Client) CreateLibraryPlaylist(name, description string, catalogSongIDs []string) (string, error) {
	attrs := map[string]any{"name": name}
	if description != "" {
		attrs["description"] = description
	}
	body := map[string]any{"attributes": attrs}
	if data := catalogTrackData(catalogSongIDs); len(data) > 0 {
		body["relationships"] = map[string]any{
			"tracks": map[string]any{"data": data},
		}
	}

	resp, err := browserDo("POST", "/v1/me/library/playlists", url.Values{}, body, c.MediaUserToken)
	if err != nil {
		return "", err
	}
	var wrapper struct {
		Data []Resource `json:"data"`
	}
	if err := json.Unmarshal(resp, &wrapper); err != nil {
		return "", fmt.Errorf("parsing create playlist response: %w", err)
	}
	if len(wrapper.Data) == 0 {
		return "", fmt.Errorf("create playlist returned no data")
	}
	return wrapper.Data[0].ID, nil
}

// add new tracks to an existing playlist (only catalog songs, not library songs)
func (c *Client) AddTracksToLibraryPlaylist(playlistID string, catalogSongIDs []string) error {
	data := catalogTrackData(catalogSongIDs)
	if len(data) == 0 {
		return nil
	}
	body := map[string]any{"data": data}
	_, err := browserDo("POST", fmt.Sprintf("/v1/me/library/playlists/%s/tracks", playlistID), url.Values{}, body, c.MediaUserToken)
	return err
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

func (c *Client) GetSongs(ids []string) ([]Resource, error) {
	var all []Resource
	for start := 0; start < len(ids); start += 100 {
		end := min(start+100, len(ids))
		params := url.Values{
			"ids":     {strings.Join(ids[start:end], ",")},
			"include": {"albums"},
		}
		data, err := c.get(fmt.Sprintf("/v1/catalog/%s/songs", c.Storefront), params)
		if err != nil {
			return all, err
		}
		var page ResourceList
		if err := json.Unmarshal(data, &page); err != nil {
			return all, fmt.Errorf("parsing songs response: %w", err)
		}
		all = append(all, page.Data...)
	}
	return all, nil
}

func (c *Client) GetSong(id string) (*Resource, error) {
	songs, err := c.GetSongs([]string{id})
	if err != nil {
		return nil, err
	}
	if len(songs) == 0 {
		return nil, fmt.Errorf("song %s not found", id)
	}
	return &songs[0], nil
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
	artist := &wrapper.Data[0]

	if albums := artist.Relationships.Albums; albums != nil {
		for next := albums.Next; next != ""; {
			u, err := url.Parse(next)
			if err != nil {
				return nil, err
			}
			pageParams := u.Query()
			pageParams.Set("limit", "100")
			data, err := c.get(u.Path, pageParams)
			if err != nil {
				return nil, err
			}
			var page ResourceList
			if err := json.Unmarshal(data, &page); err != nil {
				return nil, fmt.Errorf("parsing artist albums response: %w", err)
			}
			albums.Data = append(albums.Data, page.Data...)
			next = page.Next
		}
		albums.Next = ""
	}

	return artist, nil
}

func (c *Client) GetArtistInfo(id string, similar int) (*Resource, error) {
	params := url.Values{
		"extend":                         {"artistBio"},
		"views":                          {"similar-artists"},
		"limit[artists:similar-artists]": {strconv.Itoa(similar)},
	}
	data, err := c.get(fmt.Sprintf("/v1/catalog/%s/artists/%s", c.Storefront, id), params)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Data []Resource `json:"data"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("parsing artist info response: %w", err)
	}
	if len(wrapper.Data) == 0 {
		return nil, fmt.Errorf("artist %s not found", id)
	}
	return &wrapper.Data[0], nil
}

func (c *Client) GetArtistTopSongs(id string, limit int) ([]Resource, error) {
	params := url.Values{
		"limit":          {strconv.Itoa(limit)},
		"include[songs]": {"albums"},
	}
	data, err := c.get(fmt.Sprintf("/v1/catalog/%s/artists/%s/view/top-songs", c.Storefront, id), params)
	if err != nil {
		return nil, err
	}
	var page ResourceList
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("parsing top songs response: %w", err)
	}
	return page.Data, nil
}

func (c *Client) FindArtistID(name string) (string, error) {
	params := url.Values{
		"term":  {name},
		"types": {"artists"},
		"limit": {"25"},
	}
	data, err := c.get(fmt.Sprintf("/v1/catalog/%s/search", c.Storefront), params)
	if err != nil {
		return "", err
	}
	var wrapper struct {
		Results SearchResults `json:"results"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return "", fmt.Errorf("parsing artist search response: %w", err)
	}
	if wrapper.Results.Artists == nil {
		return "", nil
	}
	want := ArtistIDFromName(name)
	for _, a := range wrapper.Results.Artists.Data {
		if ArtistIDFromName(a.Attributes.Name) == want {
			return a.ID, nil
		}
	}
	return "", nil
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
