package serve

import (
	"fmt"
	"net/http"

	"github.com/DwifteJB/aplsonic/src/config"
	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/serve/subsonic"
	"github.com/go-chi/chi/v5"

	middleware "github.com/go-chi/chi/v5/middleware"
)

func Serve() {
	dsn := config.GenerateDSN()
	if err := db.Connect(dsn); err != nil {
		panic(err)
	}

	r := chi.NewRouter()

	r.Use(middleware.Logger)

	// ALL SUPPORTED ROUTES!!
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
	}

	// HANDLE THE ROUTES & .view!!!
	for _, route := range routes {
		r.Get(route.path, route.handler)
		r.Post(route.path, route.handler)
		// some clients use .view, idk why
		r.Get(route.path+".view", route.handler)
		r.Post(route.path+".view", route.handler)
	}

	// HANDLE NOT SUPPORTED ROUTES!!!
	for _, path := range subsonic.NotSupportedRoutes {
		r.Get(path, subsonic.NotSupported)
		r.Post(path, subsonic.NotSupported)
		r.Get(path+".view", subsonic.NotSupported)
		r.Post(path+".view", subsonic.NotSupported)
	}

	fmt.Printf("Starting server on port %d...\n", config.AppConfig.Port)
	http.ListenAndServe(fmt.Sprintf(":%d", config.AppConfig.Port), r)
}
