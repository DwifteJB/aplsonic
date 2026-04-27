package subsonic

import "net/http"

// handles routes that are not supported by aplsonic. Always returns a 200 OK with a subsonic error code and message in the body.
func NotSupported(w http.ResponseWriter, r *http.Request) {
	if _, code, msg := Authenticate(r); code != 0 {
		Fail(w, r, code, msg)
		return
	}
	Fail(w, r, 0, "This feature is not supported by aplsonic.")
}

// routes that are NOT supported!!
var NotSupportedRoutes = []string{
	// Chat
	"/rest/addChatMessage",
	"/rest/getChatMessages",

	// Internet Radio
	"/rest/createInternetRadioStation",
	"/rest/deleteInternetRadioStation",
	"/rest/getInternetRadioStations",
	"/rest/updateInternetRadioStation",

	// Podcasts
	"/rest/createPodcastChannel",
	"/rest/deletePodcastChannel",
	"/rest/deletePodcastEpisode",
	"/rest/downloadPodcastEpisode",
	"/rest/getNewestPodcasts",
	"/rest/getPodcastEpisode",
	"/rest/getPodcasts",
	"/rest/refreshPodcasts",
	 
}
