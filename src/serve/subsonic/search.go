package subsonic

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/DwifteJB/aplsonic/src/applemusic"
	"github.com/DwifteJB/aplsonic/src/config"
	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
)

// using apple music API
func Search3(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	q := r.URL.Query().Get("query")
	if q == "" {
		Fail(w, r, 10, "Required parameter 'query' is missing.")
		return
	}

	albumCount := intParam(r, "albumCount", 20)
	songCount := intParam(r, "songCount", 20)
	limit := max(albumCount, songCount)
	if limit > 50 {
		limit = 50
	}

	// use the authenticated user's Apple Music cookies.
	client, err := applemusic.NewClientFromCookies(user.AppleCookies)
	if err != nil {
		Fail(w, r, 0, "Apple Music client error: "+err.Error())
		return
	}

	results, err := client.Search(q, limit)
	if err != nil {
		Fail(w, r, 0, "Apple Music search failed: "+err.Error())
		return
	}

	fmt.Printf("Search query: %s, found %d albums and %d songs\n", q, len(results.Albums.Data), len(results.Songs.Data))

	var albumBodies []AlbumID3Body
	var songBodies []ChildBody

	if config.AppConfig.SyncOnSearch {
		applemusic.SyncSearchResults(results)

		var albums []schema.Album
		var songs []schema.Song

		if results.Albums != nil {
			ids := make([]string, 0, len(results.Albums.Data))
			for _, res := range results.Albums.Data {
				ids = append(ids, res.ID)
			}
			db.DB.Where("id IN ?", ids).Limit(albumCount).Find(&albums)
		}
		if results.Songs != nil {
			ids := make([]string, 0, len(results.Songs.Data))
			for _, res := range results.Songs.Data {
				ids = append(ids, res.ID)
			}
			db.DB.Where("id IN ?", ids).Limit(songCount).Find(&songs)
		}

		albumBodies = make([]AlbumID3Body, len(albums))
		for i, a := range albums {
			albumBodies[i] = albumToID3(a)
		}
		songBodies = make([]ChildBody, len(songs))
		for i, s := range songs {
			songBodies[i] = songToChild(s)
		}
	} else {
		if results.Albums != nil {
			n := len(results.Albums.Data)
			if n > albumCount {
				n = albumCount
			}
			albumBodies = make([]AlbumID3Body, n)
			for i, res := range results.Albums.Data[:n] {
				albumBodies[i] = appleAlbumToID3(res)
			}
		}
		if results.Songs != nil {
			n := len(results.Songs.Data)
			if n > songCount {
				n = songCount
			}
			songBodies = make([]ChildBody, n)
			for i, res := range results.Songs.Data[:n] {
				songBodies[i] = appleSongToChild(res)
			}
		}
	}

	OK(w, r, func(resp *response) {
		resp.SearchResult3 = &SearchResult3Body{
			Album: albumBodies,
			Song:  songBodies,
		}
	})
}

func intParam(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return def
	}
	return n
}

func albumToID3(a schema.Album) AlbumID3Body {
	return AlbumID3Body{
		ID:        a.ID,
		Name:      a.Name,
		Artist:    a.Artist,
		ArtistID:  a.ArtistID,
		CoverArt:  a.CoverArt,
		SongCount: a.SongCount,
		Duration:  a.Duration,
		Year:      a.Year,
		Genre:     a.Genre,
		Created:   a.CreatedAt.Format(time.RFC3339),
	}
}

func songToChild(s schema.Song) ChildBody {
	return ChildBody{
		ID:         s.ID,
		IsDir:      false,
		Title:      s.Title,
		Album:      s.Album,
		AlbumID:    s.AlbumID,
		Artist:     s.Artist,
		ArtistID:   s.ArtistID,
		Track:      s.Track,
		DiscNumber: s.DiscNumber,
		Year:       s.Year,
		Genre:      s.Genre,
		CoverArt:   s.CoverArt,
		Duration:   s.Duration,
		Type:       "music",
	}
}

func appleAlbumToID3(r applemusic.Resource) AlbumID3Body {
	body := AlbumID3Body{
		ID:        r.ID,
		Name:      r.Attributes.Name,
		Artist:    r.Attributes.ArtistName,
		SongCount: r.Attributes.TrackCount,
		Duration:  int(r.Attributes.DurationInMillis / 1000),
		Created:   time.Now().Format(time.RFC3339),
	}
	if r.Attributes.Artwork != nil {
		body.CoverArt = applemusic.FormatArtworkURL(r.Attributes.Artwork.URL)
	}
	if len(r.Attributes.GenreNames) > 0 {
		body.Genre = r.Attributes.GenreNames[0]
	}
	if r.Attributes.ReleaseDate != "" && len(r.Attributes.ReleaseDate) >= 4 {
		y := 0
		fmt.Sscanf(r.Attributes.ReleaseDate[:4], "%d", &y)
		body.Year = y
	}
	return body
}

func appleSongToChild(r applemusic.Resource) ChildBody {
	body := ChildBody{
		ID:         r.ID,
		IsDir:      false,
		Title:      r.Attributes.Name,
		Artist:     r.Attributes.ArtistName,
		Album:      r.Attributes.AlbumName,
		Track:      r.Attributes.TrackNumber,
		DiscNumber: r.Attributes.DiscNumber,
		Duration:   int(r.Attributes.DurationInMillis / 1000),
		Type:       "music",
	}
	if r.Attributes.Artwork != nil {
		body.CoverArt = applemusic.FormatArtworkURL(r.Attributes.Artwork.URL)
	}
	if len(r.Attributes.GenreNames) > 0 {
		body.Genre = r.Attributes.GenreNames[0]
	}
	if r.Attributes.ReleaseDate != "" && len(r.Attributes.ReleaseDate) >= 4 {
		y := 0
		fmt.Sscanf(r.Attributes.ReleaseDate[:4], "%d", &y)
		body.Year = y
	}
	if r.Relationships.Albums != nil && len(r.Relationships.Albums.Data) > 0 {
		body.AlbumID = r.Relationships.Albums.Data[0].ID
	}
	return body
}
