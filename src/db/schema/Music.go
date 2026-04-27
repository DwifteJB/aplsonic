package schema

import (
	"time"

	"gorm.io/gorm"
)

type Artist struct {
	ID        string         `json:"id" xml:"id" gorm:"type:varchar(191);primaryKey"`
	CreatedAt time.Time      `json:"-" xml:"-"`
	UpdatedAt time.Time      `json:"-" xml:"-"`
	DeletedAt gorm.DeletedAt `json:"-" xml:"-" gorm:"index"`

	Name     string `json:"name" xml:"name" gorm:"type:varchar(191);index"`
	SortName string `json:"sortName,omitempty" xml:"sortName,omitempty"`
	CoverArt string `json:"coverArt,omitempty" xml:"coverArt,omitempty"`

	AlbumCount int `json:"albumCount" xml:"albumCount"`
}

type Album struct {
	ID        string         `json:"id" xml:"id" gorm:"type:varchar(191);primaryKey"`
	CreatedAt time.Time      `json:"created" xml:"created"`
	UpdatedAt time.Time      `json:"-" xml:"-"`
	DeletedAt gorm.DeletedAt `json:"-" xml:"-" gorm:"index"`

	Name          string `json:"name" xml:"name" gorm:"type:varchar(191);index"`
	SortName      string `json:"sortName,omitempty" xml:"sortName,omitempty"`
	Version       string `json:"version,omitempty" xml:"version,omitempty"`
	ArtistID      string `json:"artistId,omitempty" xml:"artistId,omitempty" gorm:"type:varchar(191);index"`
	Artist        string `json:"artist,omitempty" xml:"artist,omitempty"`
	DisplayArtist string `json:"displayArtist,omitempty" xml:"displayArtist,omitempty"`
	CoverArt      string `json:"coverArt,omitempty" xml:"coverArt,omitempty"`

	SongCount int        `json:"songCount" xml:"songCount"`
	Duration  int        `json:"duration" xml:"duration"`
	PlayCount int64      `json:"playCount,omitempty" xml:"playCount,omitempty"`
	Played    *time.Time `json:"played,omitempty" xml:"played,omitempty"`

	Year  int    `json:"year,omitempty" xml:"year,omitempty"`
	Genre string `json:"genre,omitempty" xml:"genre,omitempty"`

	// release comps
	ReleaseYear  int `json:"releaseYear,omitempty" xml:"releaseYear,omitempty"`
	ReleaseMonth int `json:"releaseMonth,omitempty" xml:"releaseMonth,omitempty"`
	ReleaseDay   int `json:"releaseDay,omitempty" xml:"releaseDay,omitempty"`

	OriginalReleaseYear  int `json:"originalReleaseYear,omitempty" xml:"originalReleaseYear,omitempty"`
	OriginalReleaseMonth int `json:"originalReleaseMonth,omitempty" xml:"originalReleaseMonth,omitempty"`
	OriginalReleaseDay   int `json:"originalReleaseDay,omitempty" xml:"originalReleaseDay,omitempty"`

	IsCompilation  bool   `json:"isCompilation,omitempty" xml:"isCompilation,omitempty"`
	ExplicitStatus string `json:"explicitStatus,omitempty" xml:"explicitStatus,omitempty"`

	// json-encoded arrays stored as text
	ReleaseTypes string `json:"-" xml:"-" gorm:"type:text"` // ["Album","Remaster",...]
	Moods        string `json:"-" xml:"-" gorm:"type:text"`
	RecordLabels string `json:"-" xml:"-" gorm:"type:text"`
	DiscTitles   string `json:"-" xml:"-" gorm:"type:text"`
}

type AlbumArtist struct {
	AlbumID  string `gorm:"type:varchar(191);primaryKey;index"`
	ArtistID string `gorm:"type:varchar(191);primaryKey;index"`
}

type AlbumGenre struct {
	AlbumID string `gorm:"type:varchar(191);primaryKey;index"`
	GenreID uint   `gorm:"primaryKey;index"`
}

type Song struct {
	ID        string         `json:"id" xml:"id" gorm:"type:varchar(191);primaryKey"`
	CreatedAt time.Time      `json:"created" xml:"created"`
	UpdatedAt time.Time      `json:"-" xml:"-"`
	DeletedAt gorm.DeletedAt `json:"-" xml:"-" gorm:"index"`

	ParentID string `json:"parent,omitempty" xml:"parent,omitempty" gorm:"type:varchar(191);index"`
	Title    string `json:"title" xml:"title" gorm:"type:varchar(191);index"`
	SortName string `json:"sortName,omitempty" xml:"sortName,omitempty"`

	AlbumID string `json:"albumId,omitempty" xml:"albumId,omitempty" gorm:"type:varchar(191);index"`
	Album   string `json:"album,omitempty" xml:"album,omitempty"`

	ArtistID           string `json:"artistId,omitempty" xml:"artistId,omitempty" gorm:"type:varchar(191);index"`
	Artist             string `json:"artist,omitempty" xml:"artist,omitempty"`
	DisplayArtist      string `json:"displayArtist,omitempty" xml:"displayArtist,omitempty"`
	DisplayAlbumArtist string `json:"displayAlbumArtist,omitempty" xml:"displayAlbumArtist,omitempty"`
	DisplayComposer    string `json:"displayComposer,omitempty" xml:"displayComposer,omitempty"`

	Track      int    `json:"track,omitempty" xml:"track,omitempty"`
	DiscNumber int    `json:"discNumber,omitempty" xml:"discNumber,omitempty"`
	Year       int    `json:"year,omitempty" xml:"year,omitempty"`
	Genre      string `json:"genre,omitempty" xml:"genre,omitempty"`

	CoverArt    string `json:"coverArt,omitempty" xml:"coverArt,omitempty"`
	Size        int64  `json:"size,omitempty" xml:"size,omitempty"`
	ContentType string `json:"contentType,omitempty" xml:"contentType,omitempty"`
	Suffix      string `json:"suffix,omitempty" xml:"suffix,omitempty"`

	Duration     int `json:"duration,omitempty" xml:"duration,omitempty"`
	BitRate      int `json:"bitRate,omitempty" xml:"bitRate,omitempty"`
	BitDepth     int `json:"bitDepth,omitempty" xml:"bitDepth,omitempty"`
	SamplingRate int `json:"samplingRate,omitempty" xml:"samplingRate,omitempty"`
	ChannelCount int `json:"channelCount,omitempty" xml:"channelCount,omitempty"`

	// Filename relative to ./music directory
	Path string `json:"path,omitempty" xml:"path,omitempty"`

	PlayCount int64      `json:"playCount,omitempty" xml:"playCount,omitempty"`
	Played    *time.Time `json:"played,omitempty" xml:"played,omitempty"`

	BPM            int    `json:"bpm,omitempty" xml:"bpm,omitempty"`
	Comment        string `json:"comment,omitempty" xml:"comment,omitempty"`
	ISRC           string `json:"isrc,omitempty" xml:"isrc,omitempty"`
	ExplicitStatus string `json:"explicitStatus,omitempty" xml:"explicitStatus,omitempty"`

	// ReplayGain stored as individual columns
	ReplayGainTrackGain *float64 `json:"-" xml:"-"`
	ReplayGainTrackPeak *float64 `json:"-" xml:"-"`
	ReplayGainAlbumGain *float64 `json:"-" xml:"-"`
	ReplayGainAlbumPeak *float64 `json:"-" xml:"-"`
	ReplayGainBaseGain  *float64 `json:"-" xml:"-"`

	// JSON-encoded arrays stored as text
	Moods string `json:"-" xml:"-" gorm:"type:text"`
}

type SongArtist struct {
	SongID   string `gorm:"type:varchar(191);primaryKey;index"`
	ArtistID string `gorm:"type:varchar(191);primaryKey;index"`
	Role     string `gorm:"type:varchar(191);primaryKey"` // artist, albumartist, composer, lyricist, etc.
}

type SongGenre struct {
	SongID  string `gorm:"type:varchar(191);primaryKey;index"`
	GenreID uint   `gorm:"primaryKey;index"`
}

type Genre struct {
	ID         uint   `gorm:"primaryKey;autoIncrement" json:"-" xml:"-"`
	Name       string `gorm:"type:varchar(191);uniqueIndex" json:"value" xml:"value"`
	SongCount  int    `json:"songCount" xml:"songCount"`
	AlbumCount int    `json:"albumCount" xml:"albumCount"`
}

type Starred struct {
	Username  string    `gorm:"type:varchar(191);primaryKey;index"`
	ItemID    string    `gorm:"type:varchar(191);primaryKey"`
	ItemType  string    `gorm:"type:varchar(191);primaryKey"` // song, album, artist
	StarredAt time.Time `json:"starred" xml:"starred"`
}

type UserRating struct {
	Username string `gorm:"type:varchar(191);primaryKey;index"`
	ItemID   string `gorm:"type:varchar(191);primaryKey"`
	ItemType string `gorm:"type:varchar(191);primaryKey"` // song, album
	Rating   int    `json:"userRating" xml:"userRating"`
}

func init() {
	AllModels = append(AllModels,
		&Artist{},
		&Album{},
		&AlbumArtist{},
		&AlbumGenre{},
		&Song{},
		&SongArtist{},
		&SongGenre{},
		&Genre{},
		&Starred{},
		&UserRating{},
	)
}