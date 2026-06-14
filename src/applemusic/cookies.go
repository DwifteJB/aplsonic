package applemusic

import (
	"fmt"
	"strconv"
	"strings"
)

type Cookie struct {
	Domain     string
	IncludeSub bool
	Path       string
	Secure     bool
	Expires    int64
	Name       string
	Value      string
}

func ParseNetscape(netscape string) []Cookie {
	var out []Cookie
	for _, line := range strings.Split(netscape, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 7)
		if len(parts) < 7 {
			continue
		}
		exp, _ := strconv.ParseInt(parts[4], 10, 64)
		out = append(out, Cookie{
			Domain:     parts[0],
			IncludeSub: strings.EqualFold(parts[1], "TRUE"),
			Path:       parts[2],
			Secure:     strings.EqualFold(parts[3], "TRUE"),
			Expires:    exp,
			Name:       parts[5],
			Value:      parts[6],
		})
	}
	return out
}

func RenderNetscape(cookies []Cookie) string {
	var sb strings.Builder
	sb.WriteString("# Netscape HTTP Cookie File\n")
	for _, c := range cookies {
		includeSub := "FALSE"
		if c.IncludeSub || strings.HasPrefix(c.Domain, ".") {
			includeSub = "TRUE"
		}
		secure := "FALSE"
		if c.Secure {
			secure = "TRUE"
		}
		fmt.Fprintf(&sb, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			c.Domain, includeSub, c.Path, secure, c.Expires, c.Name, c.Value)
	}
	return sb.String()
}

func HasCookie(netscape, name string) bool {
	for _, c := range ParseNetscape(netscape) {
		if c.Name == name {
			return true
		}
	}
	return false
}
