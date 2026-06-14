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

type storageConfig struct {
	Endpoint      string `yaml:"endpoint"`       // s3 endpoint, e.g. localhost:7070
	Region        string `yaml:"region"`        // s3 region, e.g. us-east-1
	AccessKey     string `yaml:"access_key"`
	SecretKey     string `yaml:"secret_key"`
	Bucket        string `yaml:"bucket"`
	UseSSL        bool   `yaml:"use_ssl"`        // false for local versitygw
	DownloadCodec string `yaml:"download_codec"` // gamdl --song-codec-priority
}

type Config struct {
	Database dbConfig `yaml:"database"`

	Port         int    `yaml:"port"`
	WebPort      int    `yaml:"web_port"` // admin panel port; 0 or == Port serves it on the main port

	SyncOnSearch bool          `yaml:"sync_on_search"`
	Download     string        `yaml:"download"` // "getAlbum", "play", or "playAlbum"
	Storage      storageConfig `yaml:"storage"`

	// experimental token monitor settings
	TokenCheckHours int  `yaml:"token_check_hours"` // how often to re-validate tokens
	TokenWarnDays   int  `yaml:"token_warn_days"`   // flag tokens expiring within this many days
	TokenAutoRenew  bool `yaml:"token_auto_renew"`  // try a headless-browser renew for expiring tokens with a myacinfo cookie
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
