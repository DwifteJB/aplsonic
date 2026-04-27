package schema

import (
	"time"

	"gorm.io/gorm"
)

type Share struct {
	ID        string         `json:"id" xml:"id" gorm:"type:varchar(191);primaryKey"`
	CreatedAt time.Time      `json:"created" xml:"created"`
	UpdatedAt time.Time      `json:"-" xml:"-"`
	DeletedAt gorm.DeletedAt `json:"-" xml:"-" gorm:"index"`

	URL         string     `json:"url" xml:"url" gorm:"type:varchar(191);uniqueIndex"`
	Description string     `json:"description,omitempty" xml:"description,omitempty"`
	Username    string     `json:"username" xml:"username" gorm:"type:varchar(191);index"`
	ExpiresAt   *time.Time `json:"expires,omitempty" xml:"expires,omitempty"`
	LastVisited *time.Time `json:"lastVisited,omitempty" xml:"lastVisited,omitempty"`
	VisitCount  int        `json:"visitCount" xml:"visitCount" gorm:"default:0"`
}

type ShareEntry struct {
	ID      uint   `gorm:"primaryKey;autoIncrement" json:"-" xml:"-"`
	ShareID string `gorm:"type:varchar(191);index" json:"-" xml:"-"`
	SongID  string `json:"-" xml:"-"`
}

type Bookmark struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"-" xml:"-"`
	CreatedAt time.Time      `json:"created" xml:"created"`
	UpdatedAt time.Time      `json:"changed" xml:"changed"`
	DeletedAt gorm.DeletedAt `json:"-" xml:"-" gorm:"index"`

	Username string `json:"username" xml:"username" gorm:"type:varchar(191);uniqueIndex:idx_bookmark_user_song"`
	SongID   string `json:"-" xml:"-" gorm:"type:varchar(191);uniqueIndex:idx_bookmark_user_song"`
	Position int64  `json:"position" xml:"position"` // milliseconds
	Comment  string `json:"comment,omitempty" xml:"comment,omitempty"`
}

func init() {
	AllModels = append(AllModels,
		&Share{},
		&ShareEntry{},
		&Bookmark{},
	)
}