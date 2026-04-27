package config

import (
	"bytes"
	_ "embed"
	"io"
	"os"
	"strconv"

	"github.com/goccy/go-yaml"
)

//go:embed default-config.yml
var defaultConfig []byte

type dbConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type Config struct {
	Database dbConfig `yaml:"database"`

	Port         int    `yaml:"port"`
	AlbumArtPath string `yaml:"album_art_path"`
	MusicPath    string `yaml:"music_path"`

	SyncOnSearch bool   `yaml:"sync_on_search"`
	Download     string `yaml:"download"` // "getAlbum" or "play"
}

var AppConfig *Config

func GenerateDSN() string {
	// root:@tcp(tidb:4000)/test?charset=utf8mb4&parseTime=True&loc=Local
	d := AppConfig.Database
	return d.User + ":" + d.Password + "@tcp(" + d.Host + ":" + strconv.Itoa(d.Port) + ")/" + d.Database + "?charset=utf8mb4&parseTime=True&loc=Local"
}

func init() {
	config, err := os.ReadFile("configuration.yml")
	if err != nil {
		config = defaultConfig
	}

	r := io.NopCloser(bytes.NewReader(config))

	AppConfig = &Config{}
	if err = yaml.NewDecoder(r).Decode(AppConfig); err != nil {
		panic(err)
	}

	println("loaded configuration")
}
