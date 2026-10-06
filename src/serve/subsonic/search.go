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
	artistCount := intParam(r, "artistCount", 20)
	limit := max(albumCount, songCount, artistCount)
	if limit > 50 {
		limit = 50
	}
	if limit == 0 {
		OK(w, r, func(resp *response) {
			resp.SearchResult3 = &SearchResult3Body{}
		})
		return
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

	appleAlbums := firstResources(results.Albums, albumCount)
	appleSongs := firstResources(results.Songs, songCount)
	appleArtists := firstResources(results.Artists, artistCount)

	for _, res := range appleArtists {
		rememberAppleArtist(res)
	}

	fmt.Printf("Search query: %s, found %d albums, %d songs and %d artists\n", q, len(appleAlbums), len(appleSongs), len(appleArtists))

	var albumBodies []AlbumID3Body
	var songBodies []ChildBody
	var artistBodies []ArtistID3Body

	if config.AppConfig.SyncOnSearch {
		applemusic.SyncSearchResults(results)

		var albums []schema.Album
		var songs []schema.Song
		var artists []schema.Artist

		if len(appleAlbums) > 0 {
			db.DB.Where("id IN ?", resourceIDs(appleAlbums)).Find(&albums)
		}
		if len(appleSongs) > 0 {
			db.DB.Where("id IN ?", resourceIDs(appleSongs)).Find(&songs)
		}
		if len(appleArtists) > 0 {
			db.DB.Where("id IN ?", resourceIDs(appleArtists)).Find(&artists)
		}

		albumBodies = make([]AlbumID3Body, len(albums))
		for i, a := range albums {
			albumBodies[i] = albumToID3(a)
		}
		songBodies = make([]ChildBody, len(songs))
		for i, s := range songs {
			songBodies[i] = songToChild(s)
		}
		artistBodies = make([]ArtistID3Body, len(artists))
		for i, a := range artists {
			artistBodies[i] = ArtistID3Body{
				ID:             a.ID,
				Name:           a.Name,
				CoverArt:       a.CoverArt,
				AlbumCount:     a.AlbumCount,
				ArtistImageURL: a.CoverArt,
				SortName:       a.SortName,
			}
		}
	} else {
		albumBodies = make([]AlbumID3Body, len(appleAlbums))
		for i, res := range appleAlbums {
			albumBodies[i] = appleAlbumToID3(res)
		}
		songBodies = make([]ChildBody, len(appleSongs))
		for i, res := range appleSongs {
			songBodies[i] = appleSongToChild(res)
		}
		artistBodies = make([]ArtistID3Body, len(appleArtists))
		for i, res := range appleArtists {
			artistBodies[i] = appleArtistToID3(res)
		}
	}

	stars := loadStars(user.Username)
	stars.markAlbums(albumBodies)
	stars.markChildren(songBodies)
	for i := range artistBodies {
		artistBodies[i].Starred = stars["artist:"+artistBodies[i].ID]
	}

	OK(w, r, func(resp *response) {
		resp.SearchResult3 = &SearchResult3Body{
			Artist: artistBodies,
			Album:  albumBodies,
			Song:   songBodies,
		}
	})
}

func firstResources(list *applemusic.ResourceList, n int) []applemusic.Resource {
	if list == nil {
		return nil
	}
	if len(list.Data) > n {
		return list.Data[:n]
	}
	return list.Data
}

func resourceIDs(resources []applemusic.Resource) []string {
	ids := make([]string, len(resources))
	for i, res := range resources {
		ids[i] = res.ID
	}
	return ids
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

		ExplicitStatus: a.ExplicitStatus,
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

		ExplicitStatus: s.ExplicitStatus,
	}
}

func appleAlbumToID3(r applemusic.Resource) AlbumID3Body {
	body := AlbumID3Body{
		ID:        r.ID,
		Name:      r.Attributes.Name,
		Artist:    r.Attributes.ArtistName,
		ArtistID:  artistRef(r.Attributes.ArtistName),
		SongCount: r.Attributes.TrackCount,
		Duration:  int(r.Attributes.DurationInMillis / 1000),
		Created:   time.Now().Format(time.RFC3339),

		ExplicitStatus: r.Attributes.ContentRating,
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

func appleArtistToID3(r applemusic.Resource) ArtistID3Body {
	body := ArtistID3Body{
		ID:   r.ID,
		Name: r.Attributes.Name,
	}
	if r.Attributes.Artwork != nil {
		body.CoverArt = applemusic.FormatArtworkURL(r.Attributes.Artwork.URL)
		body.ArtistImageURL = body.CoverArt
	}
	return body
}

func appleSongToChild(r applemusic.Resource) ChildBody {
	body := ChildBody{
		ID:         r.ID,
		IsDir:      false,
		Title:      r.Attributes.Name,
		Artist:     r.Attributes.ArtistName,
		ArtistID:   artistRef(r.Attributes.ArtistName),
		Album:      r.Attributes.AlbumName,
		Track:      r.Attributes.TrackNumber,
		DiscNumber: r.Attributes.DiscNumber,
		Duration:   int(r.Attributes.DurationInMillis / 1000),
		Type:       "music",

		ExplicitStatus: r.Attributes.ContentRating,
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
