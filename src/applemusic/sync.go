package applemusic

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	"gorm.io/gorm/clause"

	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
)

// only if config.SyncOnSearch = true
func SyncSearchResults(results *SearchResults) {
	if results.Albums != nil {
		syncAlbums(results.Albums.Data)
	}
	if results.Songs != nil {
		syncSongs(results.Songs.Data)
	}
	if results.Artists != nil {
		syncArtists(results.Artists.Data)
	}
}

func SyncAlbum(r *Resource) {
	syncAlbums([]Resource{*r})
	if r.Relationships.Tracks != nil {
		for i := range r.Relationships.Tracks.Data {
			r.Relationships.Tracks.Data[i].Relationships.Albums = &ResourceList{
				Data: []Resource{{ID: r.ID}},
			}
		}
		syncSongs(r.Relationships.Tracks.Data)
	}
}

func SyncArtist(r *Resource) {
	syncArtists([]Resource{*r})
	if r.Relationships.Albums != nil {
		syncAlbums(r.Relationships.Albums.Data)
	}
}

func syncArtists(resources []Resource) {
	for _, r := range resources {
		artist := schema.Artist{
			ID:   r.ID,
			Name: r.Attributes.Name,
		}
		if r.Attributes.Artwork != nil {
			artist.CoverArt = FormatArtworkURL(r.Attributes.Artwork.URL)
		}
		db.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&artist)
	}
}

func syncAlbums(resources []Resource) {
	for _, r := range resources {
		album := schema.Album{
			ID:            r.ID,
			Name:          r.Attributes.Name,
			Artist:        r.Attributes.ArtistName,
			DisplayArtist: r.Attributes.ArtistName,
			SongCount:     r.Attributes.TrackCount,
			Duration:      int(r.Attributes.DurationInMillis / 1000),
			IsCompilation: r.Attributes.IsCompilation,
		}

		if len(r.Attributes.GenreNames) > 0 {
			album.Genre = r.Attributes.GenreNames[0]
		}
		if r.Attributes.Artwork != nil {
			album.CoverArt = FormatArtworkURL(r.Attributes.Artwork.URL)
		}
		if r.Attributes.ReleaseDate != "" {
			album.Year = parseYear(r.Attributes.ReleaseDate)
		}
		if r.Attributes.RecordLabel != "" {
			album.RecordLabels = r.Attributes.RecordLabel
		}

		// get a stable artist ID from the name
		if r.Attributes.ArtistName != "" {
			album.ArtistID = artistIDFromName(r.Attributes.ArtistName)
		}

		// if artist doesn't exist yet, create a stub so we have an ID to link to. This is possible because albums are returned in the search results before artists. :)
		if album.ArtistID != "" {
			artist := schema.Artist{ID: album.ArtistID, Name: r.Attributes.ArtistName}
			if album.CoverArt != "" {
				artist.CoverArt = album.CoverArt
			}
			db.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&artist)
		}

		db.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&album)
	}
}

func syncSongs(resources []Resource) {
	for _, r := range resources {
		song := schema.Song{
			ID:              r.ID,
			Title:           r.Attributes.Name,
			Album:           r.Attributes.AlbumName,
			Artist:          r.Attributes.ArtistName,
			DisplayArtist:   r.Attributes.ArtistName,
			Track:           r.Attributes.TrackNumber,
			DiscNumber:      r.Attributes.DiscNumber,
			Duration:        int(r.Attributes.DurationInMillis / 1000),
			ISRC:            r.Attributes.ISRC,
			DisplayComposer: r.Attributes.ComposerName,
			ExplicitStatus:  r.Attributes.ContentRating,
			BPM:             r.Attributes.CurrentBPM,
		}

		if len(r.Attributes.GenreNames) > 0 {
			song.Genre = r.Attributes.GenreNames[0]
		}
		if r.Attributes.Artwork != nil {
			song.CoverArt = FormatArtworkURL(r.Attributes.Artwork.URL)
		}
		if r.Attributes.ReleaseDate != "" {
			song.Year = parseYear(r.Attributes.ReleaseDate)
		}
		if r.Attributes.ArtistName != "" {
			song.ArtistID = artistIDFromName(r.Attributes.ArtistName)
		}

		// link to relationship
		if r.Relationships.Albums != nil && len(r.Relationships.Albums.Data) > 0 {
			song.AlbumID = r.Relationships.Albums.Data[0].ID
		}

		db.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&song)
	}
}

// derive 16 bit artist ID
func artistIDFromName(name string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(name))))
	return hex.EncodeToString(sum[:8])
}

func FormatArtworkURL(rawURL string) string {
	u := strings.Replace(rawURL, "{w}", "500", 1)
	u = strings.Replace(u, "{h}", "500", 1)
	return u
}

func parseYear(date string) int {
	if len(date) >= 4 {
		y, _ := strconv.Atoi(date[:4])
		return y
	}
	return 0
}

func UpdateAlbumSongCounts() {
	db.DB.Exec(`
		UPDATE albums a
		SET song_count = (SELECT COUNT(*) FROM songs s WHERE s.album_id = a.id AND s.deleted_at IS NULL)
		WHERE a.deleted_at IS NULL
	`)
}
