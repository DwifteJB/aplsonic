package schema

import "time"

// cached apple music lyrics, raw ttml so parsing changes don't need a refetch
type SongLyrics struct {
	SongID    string    `gorm:"type:varchar(191);primaryKey"`
	TTML      string    `gorm:"type:longtext"`
	Missing   bool      // apple has no lyrics for this song
	FetchedAt time.Time `gorm:"index"`
}

func init() {
	AllModels = append(AllModels, &SongLyrics{})
}
