package serve

import (
	"fmt"
	"net/http"

	"github.com/DwifteJB/aplsonic/src/applemusic"
	"github.com/DwifteJB/aplsonic/src/config"
	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/serve/admin"
	"github.com/DwifteJB/aplsonic/src/serve/subsonic"
	"github.com/DwifteJB/aplsonic/src/storage"
	"github.com/go-chi/chi/v5"

	middleware "github.com/go-chi/chi/v5/middleware"
)

func Serve() {
	dsn := config.GenerateDSN()
	if err := db.Connect(dsn); err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		panic(err)
	}

	if err := storage.Init(); err != nil {
		fmt.Printf("Failed to initialize storage: %v\n", err)
		panic(err)
	}

	if err := admin.EnsureAdmin(); err != nil {
		fmt.Printf("Failed to ensure admin user exists: %v\n", err)
		panic(err)
	}

	// admin token monitah
	admin.StartTokenMonitor()

	// keep a browser warm so the first amp-api request is fast, then import playlists
	go func() {
		applemusic.Warm()
		applemusic.SyncAllPlaylists()
	}()

	r := chi.NewRouter()

	r.Use(middleware.Logger)

	// admin panel: own port if set, else mounted on the main port
	webPort := config.AppConfig.WebPort
	if webPort == 0 || webPort == config.AppConfig.Port {
		r.Mount("/admin", admin.Router())
	} else {
		ar := chi.NewRouter()
		ar.Use(middleware.Logger)
		ar.Get("/", func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, "/admin/", http.StatusFound)
		})
		ar.Mount("/admin", admin.Router())
		go func() {
			fmt.Printf("Starting admin panel on http://localhost:%d/admin\n", webPort)
			if err := http.ListenAndServe(fmt.Sprintf(":%d", webPort), ar); err != nil {
				panic(err)
			}
		}()
	}

	// supported routes
	routes := []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/rest/ping", subsonic.Ping},
		{"/rest/getLicense", subsonic.GetLicense},
		{"/rest/getUser", subsonic.GetUser},
		{"/rest/getUsers", subsonic.GetUsers},
		{"/rest/getCoverArt", subsonic.GetCoverArt},
		{"/rest/search3", subsonic.Search3},
		{"/rest/getAlbumList", subsonic.GetAlbumList},
		{"/rest/getAlbumList2", subsonic.GetAlbumList2},
		{"/rest/getRandomSongs", subsonic.GetRandomSongs},
		{"/rest/getAlbum", subsonic.GetAlbum},
		{"/rest/getArtist", subsonic.GetArtist},
		{"/rest/getArtists", subsonic.GetArtists},
		{"/rest/getIndexes", subsonic.GetIndexes},
		{"/rest/getSong", subsonic.GetSong},
		{"/rest/stream", subsonic.Stream},
		{"/rest/download", subsonic.Download},
		{"/rest/star", subsonic.Star},
		{"/rest/unstar", subsonic.Unstar},
		{"/rest/getStarred", subsonic.GetStarred},
		{"/rest/getStarred2", subsonic.GetStarred2},
		{"/rest/getPlaylists", subsonic.GetPlaylists},
		{"/rest/getPlaylist", subsonic.GetPlaylist},
		{"/rest/createPlaylist", subsonic.CreatePlaylist},
		{"/rest/updatePlaylist", subsonic.UpdatePlaylist},
		{"/rest/deletePlaylist", subsonic.DeletePlaylist},
	}

	// register each route, plus the .view variant some clients use
	for _, route := range routes {
		r.Get(route.path, route.handler)
		r.Post(route.path, route.handler)
		r.Get(route.path+".view", route.handler)
		r.Post(route.path+".view", route.handler)
	}

	// unsupported routes reply with a subsonic error, not an http error
	for _, path := range subsonic.NotSupportedRoutes {
		r.Get(path, subsonic.NotSupported)
		r.Post(path, subsonic.NotSupported)
		r.Get(path+".view", subsonic.NotSupported)
		r.Post(path+".view", subsonic.NotSupported)
	}

	fmt.Printf("Starting server on port %d...\n", config.AppConfig.Port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", config.AppConfig.Port), r)
	if err != nil {
		fmt.Printf("Failed to start server: %v, is there something on the same port?\n", err)
		panic(err)
	}
}
