package schema

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID            uint   `json:"id" xml:"id" gorm:"primaryKey"`
	Username      string `json:"username" xml:"username" gorm:"type:varchar(191);uniqueIndex"`
	Email         string `json:"email" xml:"email" gorm:"type:varchar(191)"`
	Password      string `json:"-" xml:"-" gorm:"type:varchar(255)"`
	Salt          string `json:"-" xml:"-" gorm:"type:varchar(255)"`
	TokenPassword string `json:"-" xml:"-" gorm:"type:varchar(255)"` // plaintext, used only for MD5 token auth

	AppleCookies string `json:"-" xml:"-" gorm:"type:longtext"`

	// apple token help
	AppleTokenExpiresAt     *time.Time `json:"-" xml:"-"`
	AppleTokenStatus        string     `json:"-" xml:"-" gorm:"type:varchar(16);default:'unknown'"`
	AppleTokenLastCheckedAt *time.Time `json:"-" xml:"-"`
	AppleTokenLastError     string     `json:"-" xml:"-" gorm:"type:varchar(512)"`

	MaxBitRate        int        `json:"maxBitRate" xml:"maxBitRate" gorm:"default:0"`
	AvatarLastChanged *time.Time `json:"avatarLastChanged,omitempty" xml:"avatarLastChanged,omitempty"`

	ScrobblingEnabled   bool `json:"scrobblingEnabled" xml:"scrobblingEnabled" gorm:"default:true"`
	AdminRole           bool `json:"adminRole" xml:"adminRole" gorm:"default:false"`
	SettingsRole        bool `json:"settingsRole" xml:"settingsRole" gorm:"default:true"`
	DownloadRole        bool `json:"downloadRole" xml:"downloadRole" gorm:"default:true"`
	UploadRole          bool `json:"uploadRole" xml:"uploadRole" gorm:"default:false"`
	PlaylistRole        bool `json:"playlistRole" xml:"playlistRole" gorm:"default:true"`
	CoverArtRole        bool `json:"coverArtRole" xml:"coverArtRole" gorm:"default:true"`
	CommentRole         bool `json:"commentRole" xml:"commentRole" gorm:"default:true"`
	PodcastRole         bool `json:"podcastRole" xml:"podcastRole" gorm:"default:true"`
	StreamRole          bool `json:"streamRole" xml:"streamRole" gorm:"default:true"`
	JukeboxRole         bool `json:"jukeboxRole" xml:"jukeboxRole" gorm:"default:false"`
	ShareRole           bool `json:"shareRole" xml:"shareRole" gorm:"default:true"`
	VideoConversionRole bool `json:"videoConversionRole" xml:"videoConversionRole" gorm:"default:false"`
}

func init() {
	AllModels = append(AllModels, &User{})
}
