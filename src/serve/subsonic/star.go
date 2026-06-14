package subsonic

import (
	"fmt"
	"net/http"
	"time"

	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
	"gorm.io/gorm/clause"
)

func Star(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	items, err := starItems(r)
	if err != nil {
		Fail(w, r, 70, err.Error())
		return
	}

	now := time.Now()
	for _, it := range items {
		row := schema.Starred{
			Username:  user.Username,
			ItemID:    it.id,
			ItemType:  it.itemType,
			StarredAt: now,
		}

		if res := db.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "username"}, {Name: "item_id"}, {Name: "item_type"}},
			DoUpdates: clause.AssignmentColumns([]string{"starred_at"}),
		}).Create(&row); res.Error != nil {
			Fail(w, r, 0, res.Error.Error())
			return
		}
	}

	OK(w, r)
}

func Unstar(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	items, err := starItems(r)
	if err != nil {
		Fail(w, r, 70, err.Error())
		return
	}

	for _, it := range items {
		if res := db.DB.Where(
			"username = ? AND item_id = ? AND item_type = ?",
			user.Username, it.id, it.itemType,
		).Delete(&schema.Starred{}); res.Error != nil {
			Fail(w, r, 0, res.Error.Error())
			return
		}
	}

	OK(w, r)
}

func GetStarred(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	artists, albums, songs, err := loadStarred(user.Username)
	if err != nil {
		Fail(w, r, 0, err.Error())
		return
	}

	albumChildren := make([]ChildBody, len(albums))
	for i, a := range albums {
		albumChildren[i] = ChildBody{
			ID:       a.ID,
			IsDir:    true,
			Title:    a.Name,
			Artist:   a.Artist,
			ArtistID: a.ArtistID,
			CoverArt: a.CoverArt,
			Year:     a.Year,
			Genre:    a.Genre,
			Duration: a.Duration,
			Starred:  a.starred,
		}
	}

	OK(w, r, func(resp *response) {
		resp.Starred = &StarredBody{
			Artist: artists,
			Album:  albumChildren,
			Song:   songs,
		}
	})
}

func GetStarred2(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	artists, albums, songs, err := loadStarred(user.Username)
	if err != nil {
		Fail(w, r, 0, err.Error())
		return
	}

	albumBodies := make([]AlbumID3Body, len(albums))
	for i, a := range albums {
		body := albumToID3(a.Album)
		body.Starred = a.starred
		albumBodies[i] = body
	}

	OK(w, r, func(resp *response) {
		resp.Starred2 = &Starred2Body{
			Artist: artists,
			Album:  albumBodies,
			Song:   songs,
		}
	})
}

type starredAlbum struct {
	schema.Album
	starred string
}

func loadStarred(username string) ([]ArtistID3Body, []starredAlbum, []ChildBody, error) {
	var rows []schema.Starred
	if res := db.DB.Where("username = ?", username).Order("starred_at DESC").Find(&rows); res.Error != nil {
		return nil, nil, nil, res.Error
	}

	var albumIDs, songIDs, artistIDs []string
	starredAt := map[string]string{} 
	for _, row := range rows {
		ts := row.StarredAt.Format(time.RFC3339)
		switch row.ItemType {
		case "album":
			albumIDs = append(albumIDs, row.ItemID)
			starredAt["album:"+row.ItemID] = ts
		case "song":
			songIDs = append(songIDs, row.ItemID)
			starredAt["song:"+row.ItemID] = ts
		case "artist":
			artistIDs = append(artistIDs, row.ItemID)
			starredAt["artist:"+row.ItemID] = ts
		}
	}

	albums, err := loadStarredAlbums(albumIDs, starredAt)
	if err != nil {
		return nil, nil, nil, err
	}
	songs, err := loadStarredSongs(songIDs, starredAt)
	if err != nil {
		return nil, nil, nil, err
	}
	artists, err := loadStarredArtists(artistIDs, starredAt)
	if err != nil {
		return nil, nil, nil, err
	}
	return artists, albums, songs, nil
}

func loadStarredAlbums(ids []string, starredAt map[string]string) ([]starredAlbum, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []schema.Album
	if res := db.DB.Where("id IN ?", ids).Find(&rows); res.Error != nil {
		return nil, res.Error
	}
	byID := make(map[string]schema.Album, len(rows))
	for _, a := range rows {
		byID[a.ID] = a
	}
	out := make([]starredAlbum, 0, len(ids))
	for _, id := range ids {
		if a, ok := byID[id]; ok {
			out = append(out, starredAlbum{Album: a, starred: starredAt["album:"+id]})
		}
	}
	return out, nil
}

func loadStarredSongs(ids []string, starredAt map[string]string) ([]ChildBody, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []schema.Song
	if res := db.DB.Where("id IN ?", ids).Find(&rows); res.Error != nil {
		return nil, res.Error
	}
	byID := make(map[string]schema.Song, len(rows))
	for _, s := range rows {
		byID[s.ID] = s
	}
	out := make([]ChildBody, 0, len(ids))
	for _, id := range ids {
		if s, ok := byID[id]; ok {
			child := songToChild(s)
			child.Starred = starredAt["song:"+id]
			out = append(out, child)
		}
	}
	return out, nil
}

func loadStarredArtists(ids []string, starredAt map[string]string) ([]ArtistID3Body, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []schema.Artist
	if res := db.DB.Where("id IN ?", ids).Find(&rows); res.Error != nil {
		return nil, res.Error
	}
	byID := make(map[string]schema.Artist, len(rows))
	for _, a := range rows {
		byID[a.ID] = a
	}
	out := make([]ArtistID3Body, 0, len(ids))
	for _, id := range ids {
		if a, ok := byID[id]; ok {
			out = append(out, ArtistID3Body{
				ID:         a.ID,
				Name:       a.Name,
				CoverArt:   a.CoverArt,
				AlbumCount: a.AlbumCount,
				Starred:    starredAt["artist:"+id],
			})
		}
	}
	return out, nil
}

type starItem struct {
	id       string
	itemType string
}

func starItems(r *http.Request) ([]starItem, error) {
	q := r.URL.Query()
	var items []starItem

	for _, id := range q["albumId"] {
		if id != "" {
			items = append(items, starItem{id, "album"})
		}
	}
	for _, id := range q["artistId"] {
		if id != "" {
			items = append(items, starItem{id, "artist"})
		}
	}
	for _, id := range q["id"] {
		if id == "" {
			continue
		}
		items = append(items, starItem{id, inferType(id)})
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("required parameter 'id', 'albumId' or 'artistId' is missing")
	}
	return items, nil
}

// inferType figures out what a bare id refers to. albums and songs use Apple
// catalog ids as primary keys, artists use derived hashes — so a lookup tells
// them apart. defaults to song when unknown, matching client expectations.
func inferType(id string) string {
	var count int64
	if db.DB.Model(&schema.Album{}).Where("id = ?", id).Count(&count); count > 0 {
		return "album"
	}
	if db.DB.Model(&schema.Song{}).Where("id = ?", id).Count(&count); count > 0 {
		return "song"
	}
	if db.DB.Model(&schema.Artist{}).Where("id = ?", id).Count(&count); count > 0 {
		return "artist"
	}
	return "song"
}
