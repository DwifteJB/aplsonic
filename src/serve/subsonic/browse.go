package subsonic

import (
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/DwifteJB/aplsonic/src/applemusic"
	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
)

func GetAlbum(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		Fail(w, r, 10, "Required parameter 'id' is missing.")
		return
	}

	var album schema.Album
	albumInDB := db.DB.First(&album, "id = ?", id).Error == nil

	var songs []schema.Song
	if albumInDB {
		db.DB.Where("album_id = ?", id).Order("disc_number ASC, track ASC").Find(&songs)
	}

	if !albumInDB || len(songs) == 0 {
		client, err := applemusic.NewClientFromCookies(user.AppleCookies)
		if err != nil {
			Fail(w, r, 0, "Apple Music client error: "+err.Error())
			return
		}
		resource, err := client.GetAlbum(id)
		if err != nil {
			if !albumInDB {
				Fail(w, r, 70, "Album not found.")
				return
			}
		} else {
			applemusic.SyncAlbum(resource)
			db.DB.First(&album, "id = ?", id)
			db.DB.Where("album_id = ?", id).Order("disc_number ASC, track ASC").Find(&songs)
		}
	}

	body := albumToID3(album)
	body.Song = make([]ChildBody, len(songs))
	for i, s := range songs {
		body.Song[i] = songToChild(s)
	}

	OK(w, r, func(resp *response) {
		resp.Album = &body
	})
}

func GetArtist(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		Fail(w, r, 10, "Required parameter 'id' is missing.")
		return
	}

	var artist schema.Artist
	if res := db.DB.First(&artist, "id = ?", id); res.Error != nil {
		client, err := applemusic.NewClientFromCookies(user.AppleCookies)
		if err != nil {
			Fail(w, r, 0, "Apple Music client error: "+err.Error())
			return
		}
		resource, err := client.GetArtist(id)
		if err != nil {
			Fail(w, r, 70, "Artist not found.")
			return
		}
		applemusic.SyncArtist(resource)
		if res2 := db.DB.First(&artist, "id = ?", id); res2.Error != nil {
			Fail(w, r, 70, "Artist not found after sync.")
			return
		}
	}

	var albums []schema.Album
	db.DB.Where("artist_id = ?", id).Order("year ASC, name ASC").Find(&albums)

	albumBodies := make([]AlbumID3Body, len(albums))
	for i, a := range albums {
		albumBodies[i] = albumToID3(a)
	}

	OK(w, r, func(resp *response) {
		resp.Artist = &ArtistID3Body{
			ID:         artist.ID,
			Name:       artist.Name,
			CoverArt:   artist.CoverArt,
			AlbumCount: artist.AlbumCount,
			Album:      albumBodies,
		}
	})
}

func GetArtists(w http.ResponseWriter, r *http.Request) {
	_, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	var artists []schema.Artist
	db.DB.Order("name ASC").Find(&artists)

	indexMap := make(map[string][]ArtistID3Body)
	for _, a := range artists {
		key := indexKey(a.Name)
		indexMap[key] = append(indexMap[key], ArtistID3Body{
			ID:         a.ID,
			Name:       a.Name,
			CoverArt:   a.CoverArt,
			AlbumCount: a.AlbumCount,
		})
	}

	keys := sortedKeys(indexMap)
	indexes := make([]ArtistIndex, 0, len(keys))
	for _, k := range keys {
		indexes = append(indexes, ArtistIndex{Name: k, Artist: indexMap[k]})
	}

	OK(w, r, func(resp *response) {
		resp.Artists = &ArtistsBody{
			IgnoredArticles: "The An A Die Das Ein Les Le La",
			Index:           indexes,
		}
	})
}

func GetSong(w http.ResponseWriter, r *http.Request) {
	_, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		Fail(w, r, 10, "Required parameter 'id' is missing.")
		return
	}

	var song schema.Song
	if res := db.DB.First(&song, "id = ?", id); res.Error != nil {
		Fail(w, r, 70, "Song not found.")
		return
	}

	body := songToChild(song)
	OK(w, r, func(resp *response) {
		resp.Song = &body
	})
}

func GetIndexes(w http.ResponseWriter, r *http.Request) {
	_, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	var artists []schema.Artist
	db.DB.Order("name ASC").Find(&artists)

	indexMap := make(map[string][]ChildBody)
	for _, a := range artists {
		key := indexKey(a.Name)
		indexMap[key] = append(indexMap[key], ChildBody{
			ID:    a.ID,
			IsDir: true,
			Title: a.Name,
		})
	}

	keys := sortedChildKeys(indexMap)
	indexes := make([]IndexEntry, 0, len(keys))
	for _, k := range keys {
		indexes = append(indexes, IndexEntry{Name: k, Artist: indexMap[k]})
	}

	OK(w, r, func(resp *response) {
		resp.Indexes = &IndexesBody{
			LastModified:    time.Now().UnixMilli(),
			IgnoredArticles: "The An A Die Das Ein Les Le La",
			Index:           indexes,
		}
	})
}

// this is for ignoring certain index articles like "The", "A", "An" when sorting artists in indexes and search results
func indexKey(name string) string {
	ignored := []string{"The ", "An ", "A ", "Die ", "Das ", "Ein ", "Les ", "Le ", "La "}
	n := name
	for _, prefix := range ignored {
		if strings.HasPrefix(strings.ToLower(n), strings.ToLower(prefix)) {
			n = n[len(prefix):]
			break
		}
	}
	if len(n) == 0 {
		return "#"
	}
	r := rune(n[0])
	if unicode.IsLetter(r) {
		return strings.ToUpper(string(unicode.ToUpper(r)))
	}
	return "#"
}

// sorts via insertion sort (super simple) :)
func sortedKeys(m map[string][]ArtistID3Body) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

// sort child via insertion sort
func sortedChildKeys(m map[string][]ChildBody) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
