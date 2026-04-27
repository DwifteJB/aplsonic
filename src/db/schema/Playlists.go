package schema

import (
	"time"

	"gorm.io/gorm"
)

type Playlist struct {
	ID        string         `json:"id" xml:"id" gorm:"type:varchar(191);primaryKey"`
	CreatedAt time.Time      `json:"created" xml:"created"`
	UpdatedAt time.Time      `json:"changed" xml:"changed"`
	DeletedAt gorm.DeletedAt `json:"-" xml:"-" gorm:"index"`

	Name     string `json:"name" xml:"name"`
	Comment  string `json:"comment,omitempty" xml:"comment,omitempty"`
	Owner    string `json:"owner,omitempty" xml:"owner,omitempty" gorm:"type:varchar(191);index"`
	Public   bool   `json:"public" xml:"public" gorm:"default:false"`
	CoverArt string `json:"coverArt,omitempty" xml:"coverArt,omitempty"`
}

// playlist entry is an ordered song within a playlist
type PlaylistEntry struct {
	ID         uint   `gorm:"primaryKey;autoIncrement" json:"-" xml:"-"`
	PlaylistID string `gorm:"type:varchar(191);index" json:"-" xml:"-"`
	SongID     string `json:"-" xml:"-"`
	Position   int    `json:"-" xml:"-"`
}

// for collaborative playlists, tracks which users are allowed to edit
type PlaylistAllowedUser struct {
	PlaylistID string `gorm:"type:varchar(191);primaryKey;index" json:"-" xml:"-"`
	Username   string `gorm:"type:varchar(191);primaryKey" json:"-" xml:"-"`
}

// tracks a user's saved play queue (not the same as NowPlaying, which tracks what they're currently playing)
type PlayQueue struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"-" xml:"-"`
	CreatedAt time.Time      `json:"-" xml:"-"`
	UpdatedAt time.Time      `json:"changed" xml:"changed"`
	DeletedAt gorm.DeletedAt `json:"-" xml:"-" gorm:"index"`

	Username  string `json:"username" xml:"username" gorm:"type:varchar(191);uniqueIndex"`
	CurrentID string `json:"current,omitempty" xml:"current,omitempty"`
	Position  int64  `json:"position,omitempty" xml:"position,omitempty"` // milliseconds
	ChangedBy string `json:"changedBy" xml:"changedBy"`
}

type PlayQueueEntry struct {
	ID       uint   `gorm:"primaryKey;autoIncrement" json:"-" xml:"-"`
	Username string `gorm:"type:varchar(191);index" json:"-" xml:"-"`
	SongID   string `json:"-" xml:"-"`
	Position int    `json:"-" xml:"-"`
}

type NowPlaying struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"-" xml:"-"`
	CreatedAt time.Time      `json:"-" xml:"-"`
	UpdatedAt time.Time      `json:"-" xml:"-"`
	DeletedAt gorm.DeletedAt `json:"-" xml:"-" gorm:"index"`

	Username     string  `json:"username" xml:"username" gorm:"type:varchar(191);uniqueIndex"`
	SongID       string  `json:"id" xml:"id"`
	PlayerName   string  `json:"playerName,omitempty" xml:"playerName,omitempty"`
	PlayerID     int     `json:"playerId" xml:"playerId"`
	State        string  `json:"state,omitempty" xml:"state,omitempty"` // starting, playing, paused, stopped
	PositionMs   int64   `json:"positionMs,omitempty" xml:"positionMs,omitempty"`
	PlaybackRate float32 `json:"playbackRate,omitempty" xml:"playbackRate,omitempty"`
}

func init() {
	AllModels = append(AllModels,
		&Playlist{},
		&PlaylistEntry{},
		&PlaylistAllowedUser{},
		&PlayQueue{},
		&PlayQueueEntry{},
		&NowPlaying{},
	)
}