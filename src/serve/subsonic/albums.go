package subsonic

import (
	"fmt"
	"net/http"

	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
)

func GetAlbumList(w http.ResponseWriter, r *http.Request) {
	_, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	albums, err := queryAlbums(r)
	if err != nil {
		Fail(w, r, 0, err.Error())
		return
	}

	children := make([]ChildBody, len(albums))
	for i, a := range albums {
		children[i] = ChildBody{
			ID:       a.ID,
			IsDir:    true,
			Title:    a.Name,
			Artist:   a.Artist,
			ArtistID: a.ArtistID,
			CoverArt: a.CoverArt,
			Year:     a.Year,
			Genre:    a.Genre,
			Duration: a.Duration,
		}
	}

	OK(w, r, func(resp *response) {
		resp.AlbumList = &AlbumListBody{Album: children}
	})
}

func GetAlbumList2(w http.ResponseWriter, r *http.Request) {
	_, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	albums, err := queryAlbums(r)
	if err != nil {
		Fail(w, r, 0, err.Error())
		return
	}

	bodies := make([]AlbumID3Body, len(albums))
	for i, a := range albums {
		bodies[i] = albumToID3(a)
	}

	OK(w, r, func(resp *response) {
		resp.AlbumList2 = &AlbumList2Body{Album: bodies}
	})
}

// queryAlbums builds and executes the album query based on the type= param.
func queryAlbums(r *http.Request) ([]schema.Album, error) {
	q := r.URL.Query()
	listType := q.Get("type")
	size := intParam(r, "size", 10)
	offset := intParam(r, "offset", 0)
	if size > 500 {
		size = 500
	}

	tx := db.DB.Limit(size).Offset(offset)

	switch listType {
	case "random":
		tx = tx.Order("RAND()")

	case "newest":
		tx = tx.Order("created_at DESC")

	case "alphabeticalByName":
		tx = tx.Order("name ASC")

	case "alphabeticalByArtist":
		tx = tx.Order("artist ASC, name ASC")

	case "frequent":
		tx = tx.Order("play_count DESC")

	case "recent":
		tx = tx.Where("played IS NOT NULL").Order("played DESC")

	case "byYear":
		from := intParam(r, "fromYear", 0)
		to := intParam(r, "toYear", 9999)
		if from <= to {
			tx = tx.Where("year BETWEEN ? AND ?", from, to).Order("year ASC")
		} else {
			tx = tx.Where("year BETWEEN ? AND ?", to, from).Order("year DESC")
		}

	case "byGenre":
		genre := q.Get("genre")
		if genre == "" {
			return nil, fmt.Errorf("genre parameter required for byGenre type")
		}
		tx = tx.Where("genre = ?", genre).Order("name ASC")

	case "starred":
		tx = tx.Joins("JOIN starreds ON starreds.item_id = albums.id AND starreds.item_type = 'album'").
			Order("starreds.starred_at DESC")

	case "highest":
		// TODO: implement rating system
		tx = tx.Order("created_at DESC")

	default:
		if listType == "" {
			return nil, fmt.Errorf("required parameter 'type' is missing")
		}
		tx = tx.Order("created_at DESC")
	}

	var albums []schema.Album
	if res := tx.Find(&albums); res.Error != nil {
		return nil, res.Error
	}
	return albums, nil
}
