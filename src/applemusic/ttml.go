package applemusic

import (
	"encoding/xml"
	"math"
	"sort"
	"strconv"
	"strings"
)

// parsed apple music lyrics (TTML), kept source agnostic so the subsonic layer can shape it
type Lyrics struct {
	Lang      string
	Synced    bool // line timing at least
	WordTimed bool // itunes:timing="Word", lines have syllables

	Agents         []LyricAgent
	Lines          []LyricLine
	Translations   []LyricTrack
	Pronunciations []LyricTrack
}

type LyricAgent struct {
	ID   string
	Type string // person, group, other
	Name string
}

// translation / transliteration, lines reference the main lines by key
type LyricTrack struct {
	Lang  string
	Lines []LyricLine
}

type LyricLine struct {
	Key        string
	Agent      string
	Begin, End int64 // ms, only meaningful when synced

	Vocals     LyricVocal
	Background *LyricVocal // ttm:role="x-bg" spans
}

// one vocal layer of a line, syllable byte offsets are inclusive into Text
type LyricVocal struct {
	Text      string
	Syllables []LyricSyllable
}

type LyricSyllable struct {
	Begin, End         int64
	Text               string
	ByteStart, ByteEnd int
}

// full line text, main + background
func (l LyricLine) Text() string {
	if l.Background == nil || l.Background.Text == "" {
		return l.Vocals.Text
	}
	if l.Vocals.Text == "" {
		return l.Background.Text
	}
	return l.Vocals.Text + " " + l.Background.Text
}

// minimal dom, encoding/xml can't keep mixed content order with structs
type ttmlNode struct {
	name     string
	attrs    map[string]string // keyed by local name
	children []any             // *ttmlNode or string
}

func (n *ttmlNode) attr(local string) string {
	return n.attrs[local]
}

func (n *ttmlNode) each(local string, fn func(*ttmlNode)) {
	for _, c := range n.children {
		if child, ok := c.(*ttmlNode); ok {
			if child.name == local {
				fn(child)
			}
			child.each(local, fn)
		}
	}
}

func (n *ttmlNode) first(local string) *ttmlNode {
	var found *ttmlNode
	n.each(local, func(c *ttmlNode) {
		if found == nil {
			found = c
		}
	})
	return found
}

func (n *ttmlNode) innerText() string {
	var sb strings.Builder
	for _, c := range n.children {
		switch v := c.(type) {
		case string:
			sb.WriteString(v)
		case *ttmlNode:
			sb.WriteString(v.innerText())
		}
	}
	return sb.String()
}

func parseTTMLTree(data string) (*ttmlNode, error) {
	dec := xml.NewDecoder(strings.NewReader(data))
	root := &ttmlNode{}
	stack := []*ttmlNode{root}
	for {
		tok, err := dec.Token()
		if err != nil {
			if len(stack) == 1 && root.children != nil {
				break
			}
			return nil, err
		}
		cur := stack[len(stack)-1]
		switch t := tok.(type) {
		case xml.StartElement:
			n := &ttmlNode{name: t.Name.Local, attrs: map[string]string{}}
			for _, a := range t.Attr {
				n.attrs[a.Name.Local] = a.Value
			}
			cur.children = append(cur.children, n)
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			cur.children = append(cur.children, string(t))
		}
	}
	return root, nil
}

func ParseTTML(data string) (*Lyrics, error) {
	root, err := parseTTMLTree(data)
	if err != nil {
		return nil, err
	}
	tt := root.first("tt")
	if tt == nil {
		tt = root
	}

	lyr := &Lyrics{Lang: tt.attr("lang")}
	timing := strings.ToLower(tt.attr("timing"))
	lyr.WordTimed = timing == "word"

	if head := tt.first("head"); head != nil {
		head.each("agent", func(a *ttmlNode) {
			agent := LyricAgent{ID: a.attr("id"), Type: a.attr("type")}
			if name := a.first("name"); name != nil {
				agent.Name = strings.TrimSpace(name.innerText())
			}
			if agent.ID != "" {
				lyr.Agents = append(lyr.Agents, agent)
			}
		})
	}

	body := tt.first("body")
	if body == nil {
		return lyr, nil
	}

	synced := false
	body.each("p", func(p *ttmlNode) {
		line := parseLyricLine(p, lyr.WordTimed)
		line.Key = p.attr("key")
		line.Agent = p.attr("agent")
		if begin, ok := parseTTMLTime(p.attr("begin")); ok {
			synced = true
			line.Begin = begin
			line.End, _ = parseTTMLTime(p.attr("end"))
		}
		if line.Text() != "" {
			lyr.Lines = append(lyr.Lines, line)
		}
	})
	lyr.Synced = synced && timing != "none"
	if !lyr.Synced {
		lyr.WordTimed = false
	}
	if lyr.Synced {
		sort.SliceStable(lyr.Lines, func(i, j int) bool { return lyr.Lines[i].Begin < lyr.Lines[j].Begin })
	}

	if head := tt.first("head"); head != nil && len(lyr.Lines) > 0 {
		head.each("translation", func(t *ttmlNode) {
			if track, ok := parseLyricTrack(t, lyr, false); ok {
				lyr.Translations = append(lyr.Translations, track)
			}
		})
		head.each("transliteration", func(t *ttmlNode) {
			if track, ok := parseLyricTrack(t, lyr, true); ok {
				lyr.Pronunciations = append(lyr.Pronunciations, track)
			}
		})
	}

	return lyr, nil
}

// <text for="L1"> entries, ordered like the main lines and timed from them
func parseLyricTrack(t *ttmlNode, lyr *Lyrics, timed bool) (LyricTrack, bool) {
	texts := map[string]*ttmlNode{}
	t.each("text", func(n *ttmlNode) {
		if key := n.attr("for"); key != "" {
			texts[key] = n
		}
	})

	track := LyricTrack{Lang: t.attr("lang")}
	for _, main := range lyr.Lines {
		n, ok := texts[main.Key]
		if !ok || main.Key == "" {
			continue
		}
		line := parseLyricLine(n, timed && lyr.WordTimed)
		if line.Text() == "" {
			continue
		}
		line.Key = main.Key
		line.Agent = main.Agent
		line.Begin, line.End = main.Begin, main.End
		track.Lines = append(track.Lines, line)
	}
	return track, len(track.Lines) > 0
}

func parseLyricLine(p *ttmlNode, wordTimed bool) LyricLine {
	var main, bg vocalBuilder
	walkLyricNode(p, &main, &bg, false, wordTimed)

	line := LyricLine{Vocals: main.vocal()}
	if v := bg.vocal(); v.Text != "" {
		line.Background = &v
	}
	return line
}

func walkLyricNode(n *ttmlNode, main, bg *vocalBuilder, inBg, wordTimed bool) {
	cur := main
	if inBg {
		cur = bg
	}
	for _, c := range n.children {
		switch v := c.(type) {
		case string:
			cur.add(v, nil)
		case *ttmlNode:
			if v.attr("role") == "x-bg" {
				walkLyricNode(v, main, bg, true, wordTimed)
				continue
			}
			begin, ok := parseTTMLTime(v.attr("begin"))
			if !wordTimed || !ok {
				walkLyricNode(v, main, bg, inBg, wordTimed)
				continue
			}
			end, hasEnd := parseTTMLTime(v.attr("end"))
			if !hasEnd {
				end = -1
			}
			cur.add(v.innerText(), &LyricSyllable{Begin: begin, End: end})
		}
	}
}

// builds a whitespace-collapsed string and tracks utf-8 byte offsets for timed pieces
type vocalBuilder struct {
	sb           strings.Builder
	pendingSpace bool
	syllables    []LyricSyllable
}

func (b *vocalBuilder) add(s string, syl *LyricSyllable) {
	if s == "" {
		return
	}
	core := strings.Join(strings.Fields(s), " ")
	if core == "" {
		if b.sb.Len() > 0 {
			b.pendingSpace = true
		}
		return
	}
	if isSpace(s[0]) && b.sb.Len() > 0 {
		b.pendingSpace = true
	}
	if b.pendingSpace {
		b.sb.WriteByte(' ')
		b.pendingSpace = false
	}
	start := b.sb.Len()
	b.sb.WriteString(core)
	if syl != nil {
		syl.Text = core
		syl.ByteStart = start
		syl.ByteEnd = b.sb.Len() - 1
		b.syllables = append(b.syllables, *syl)
	}
	if isSpace(s[len(s)-1]) {
		b.pendingSpace = true
	}
}

// fills missing ends and removes overlaps so cues can be read sequentially
func (b *vocalBuilder) vocal() LyricVocal {
	syl := b.syllables
	for i := range syl {
		if syl[i].End < 0 {
			if i+1 < len(syl) {
				syl[i].End = syl[i+1].Begin
			} else {
				syl[i].End = syl[i].Begin
			}
		}
		if i+1 < len(syl) && syl[i].End > syl[i+1].Begin {
			syl[i].End = syl[i+1].Begin
		}
		if syl[i].End < syl[i].Begin {
			syl[i].End = syl[i].Begin
		}
	}
	return LyricVocal{Text: b.sb.String(), Syllables: syl}
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// handles "12.345", "1:02.345", "00:01:02.345" and "12.345s"
func parseTTMLTime(s string) (int64, bool) {
	s = strings.TrimSuffix(strings.TrimSpace(s), "s")
	if s == "" {
		return 0, false
	}
	parts := strings.Split(s, ":")
	if len(parts) > 3 {
		return 0, false
	}
	var total float64
	for _, part := range parts {
		v, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return 0, false
		}
		total = total*60 + v
	}
	return int64(math.Round(total * 1000)), true
}
