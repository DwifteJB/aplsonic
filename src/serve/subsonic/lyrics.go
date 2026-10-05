package subsonic

import (
	"net/http"
	"strings"

	"github.com/DwifteJB/aplsonic/src/applemusic"
	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
)

// GET/POST /rest/getLyricsBySongId (opensubsonic songLyrics v1 + v2 with enhanced=true)
func GetLyricsBySongID(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		Fail(w, r, 10, "Required parameter 'id' is missing.")
		return
	}
	enhanced := strings.EqualFold(r.URL.Query().Get("enhanced"), "true")

	var song schema.Song
	if res := db.DB.First(&song, "id = ?", id); res.Error != nil {
		Fail(w, r, 70, "Song not found.")
		return
	}

	lyr, err := applemusic.SongLyrics(user, song.ID)
	if err != nil {
		Fail(w, r, 0, "could not fetch lyrics: "+err.Error())
		return
	}

	list := &LyricsListBody{StructuredLyrics: []StructuredLyricsBody{}}
	if lyr != nil {
		list.StructuredLyrics = structuredLyrics(song, lyr, enhanced)
	}

	OK(w, r, func(resp *response) {
		resp.LyricsList = list
	})
}

// GET/POST /rest/getLyrics, the original subsonic artist + title lookup (plain text)
func GetLyrics(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	artist := strings.TrimSpace(r.URL.Query().Get("artist"))
	title := strings.TrimSpace(r.URL.Query().Get("title"))
	body := &LyricsBody{}

	var song schema.Song
	found := false
	if title != "" {
		tx := db.DB.Where("LOWER(title) = LOWER(?)", title)
		if artist != "" {
			tx = tx.Where("LOWER(artist) LIKE LOWER(?)", "%"+artist+"%")
		}
		tx.Limit(1).Find(&song)
		found = song.ID != ""
	}

	if found {
		lyr, err := applemusic.SongLyrics(user, song.ID)
		if err != nil {
			Fail(w, r, 0, "could not fetch lyrics: "+err.Error())
			return
		}
		if lyr != nil {
			lines := make([]string, len(lyr.Lines))
			for i, l := range lyr.Lines {
				lines[i] = l.Text()
			}
			body.Artist = songDisplayArtist(song)
			body.Title = song.Title
			body.Value = strings.Join(lines, "\n")
		}
	}

	OK(w, r, func(resp *response) {
		resp.Lyrics = body
	})
}

func songDisplayArtist(s schema.Song) string {
	if s.DisplayArtist != "" {
		return s.DisplayArtist
	}
	return s.Artist
}

// main track first, then translations and pronunciations (enhanced only)
func structuredLyrics(song schema.Song, lyr *applemusic.Lyrics, enhanced bool) []StructuredLyricsBody {
	lang := lyr.Lang
	if lang == "" {
		lang = "und"
	}

	out := []StructuredLyricsBody{
		lyricTrack(lyr.Lines, lyr.Agents, lang, "main", lyr.Synced, enhanced && lyr.WordTimed, enhanced),
	}
	if enhanced {
		for _, t := range lyr.Translations {
			out = append(out, lyricTrack(t.Lines, lyr.Agents, langOrUnd(t.Lang), "translation", lyr.Synced, false, true))
		}
		for _, t := range lyr.Pronunciations {
			out = append(out, lyricTrack(t.Lines, lyr.Agents, langOrUnd(t.Lang), "pronunciation", lyr.Synced, lyr.WordTimed, true))
		}
	}

	for i := range out {
		out[i].DisplayArtist = songDisplayArtist(song)
		out[i].DisplayTitle = song.Title
	}
	return out
}

func langOrUnd(lang string) string {
	if lang == "" {
		return "und"
	}
	return lang
}

const bgAgentID = "bg"

func lyricTrack(lines []applemusic.LyricLine, agents []applemusic.LyricAgent, lang, kind string, synced, cues, enhanced bool) StructuredLyricsBody {
	track := StructuredLyricsBody{
		Lang:   lang,
		Synced: synced,
		Line:   make([]LyricLineBody, len(lines)),
	}
	if enhanced {
		track.Kind = kind
	}
	for i, l := range lines {
		track.Line[i].Value = l.Text()
		if synced {
			start := l.Begin
			track.Line[i].Start = &start
		}
	}

	// cueLine only makes sense for synced lyrics
	if !synced || !cues {
		return track
	}

	roles, mainID := agentRoles(lines, agents)
	hasBg := false
	for _, l := range lines {
		if l.Background != nil && len(l.Background.Syllables) > 0 {
			hasBg = true
		}
	}
	// agents are only needed once there is more than one vocal layer
	withAgents := len(roles) > 1 || hasBg

	agentOf := func(l applemusic.LyricLine) string {
		if l.Agent == "" {
			return mainID
		}
		return l.Agent
	}

	for i, l := range lines {
		if cl, ok := cueLine(i, l.Vocals); ok {
			if withAgents {
				cl.AgentID = agentOf(l)
			}
			track.CueLine = append(track.CueLine, cl)
		}
		if l.Background != nil {
			if cl, ok := cueLine(i, *l.Background); ok {
				cl.AgentID = bgAgentID
				track.CueLine = append(track.CueLine, cl)
			}
		}
	}

	if withAgents && len(track.CueLine) > 0 {
		for _, a := range agents {
			if role, ok := roles[a.ID]; ok {
				track.Agents = append(track.Agents, LyricAgentBody{ID: a.ID, Role: role, Name: a.Name})
				delete(roles, a.ID)
			}
		}
		// agents referenced by lines but missing from the ttml head
		if role, ok := roles[mainID]; ok {
			track.Agents = append([]LyricAgentBody{{ID: mainID, Role: role}}, track.Agents...)
			delete(roles, mainID)
		}
		for id, role := range roles {
			track.Agents = append(track.Agents, LyricAgentBody{ID: id, Role: role})
		}
		if hasBg {
			track.Agents = append(track.Agents, LyricAgentBody{ID: bgAgentID, Role: "bg"})
		}
	}
	return track
}

// roles for the agents actually used by lines, exactly one "main"
// the first person agent in the ttml head is the lead, falling back to the first used agent
func agentRoles(lines []applemusic.LyricLine, agents []applemusic.LyricAgent) (map[string]string, string) {
	types := map[string]string{}
	for _, a := range agents {
		types[a.ID] = a.Type
	}

	mainID := ""
	for _, a := range agents {
		if a.Type == "person" {
			mainID = a.ID
			break
		}
	}
	if mainID == "" && len(agents) > 0 {
		mainID = agents[0].ID
	}
	if mainID == "" {
		mainID = "main"
	}

	used := map[string]bool{}
	var order []string
	for _, l := range lines {
		if len(l.Vocals.Syllables) == 0 {
			continue
		}
		id := l.Agent
		if id == "" {
			id = mainID
		}
		if !used[id] {
			used[id] = true
			order = append(order, id)
		}
	}

	if len(order) > 0 && !used[mainID] {
		// the lead never sings here, promote the first one that does
		mainID = order[0]
	}

	roles := map[string]string{}
	for _, id := range order {
		switch {
		case id == mainID:
			roles[id] = "main"
		case types[id] == "group":
			roles[id] = "group"
		default:
			roles[id] = "voice"
		}
	}
	return roles, mainID
}

func cueLine(index int, v applemusic.LyricVocal) (CueLineBody, bool) {
	if len(v.Syllables) == 0 {
		return CueLineBody{}, false
	}
	cl := CueLineBody{
		Index: index,
		Start: v.Syllables[0].Begin,
		End:   v.Syllables[len(v.Syllables)-1].End,
		Value: v.Text,
		Cue:   make([]CueBody, len(v.Syllables)),
	}
	for i, s := range v.Syllables {
		cl.Cue[i] = CueBody{
			Start:     s.Begin,
			End:       s.End,
			ByteStart: s.ByteStart,
			ByteEnd:   s.ByteEnd,
			Value:     s.Text,
		}
	}
	return cl, true
}

// GET/POST /rest/getOpenSubsonicExtensions, must be reachable without auth
func GetOpenSubsonicExtensions(w http.ResponseWriter, r *http.Request) {
	OK(w, r, func(resp *response) {
		resp.OpenSubsonicExtensions = []OpenSubsonicExtension{
			{Name: "songLyrics", Versions: []int{1, 2}},
		}
	})
}
