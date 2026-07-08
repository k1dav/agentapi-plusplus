package httpapi

import (
	"context"
	"regexp"
	"strings"
)

// URL extraction for GET /links. The terminal hard-wraps long URLs at the
// screen width, so URLs copied from screen-scraped messages arrive broken.
// Two sources are combined:
//   - timeline events: content comes from the agent's transcript files and
//     is never wrapped — URLs are exact
//   - conversation messages: screen-scraped; a de-wrapping heuristic rejoins
//     URLs that were split across full-width lines

// URLs are matched over printable ASCII only: terminal output decorates
// text with box-drawing and other non-ASCII glyphs (e.g. ⎿) that would
// otherwise glue onto an adjacent URL.
var urlRe = regexp.MustCompile("(?:https?://|www\\.)[^\\s\"'<>`\\\\\\x{80}-\\x{10FFFF}]+")

// urlContinuationRe matches the leading URL-safe run of a wrapped line's
// continuation.
var urlContinuationRe = regexp.MustCompile("^[^\\s\"'<>`\\\\\\x{80}-\\x{10FFFF}]+")

// trailingPunct strips characters that commonly trail a URL in prose but are
// rarely part of it.
const trailingPunct = ".,;:!?)]}>*"

// wrapWidthThreshold: a line at least this long is considered to have hit
// the terminal's right edge (default terminal width is 80 columns; message
// formatting may trim a column or two).
const wrapWidthThreshold = 78

func trimURL(url string) string {
	url = strings.TrimRight(url, trailingPunct)
	// Keep balanced closing parens (e.g. wikipedia URLs like .../Go_(lang)).
	for strings.Count(url, "(") > strings.Count(url, ")") {
		url += ")"
	}
	return url
}

// extractURLs returns all URLs in text, in order of appearance. No
// de-wrapping: intended for transcript-sourced (unwrapped) content.
func extractURLs(text string) []string {
	matches := urlRe.FindAllString(text, -1)
	urls := make([]string, 0, len(matches))
	for _, m := range matches {
		if u := trimURL(m); u != "" {
			urls = append(urls, u)
		}
	}
	return urls
}

// extractURLsFromScreen extracts URLs from screen-formatted text, rejoining
// URLs the terminal wrapped across lines: while the URL's current segment
// sits at the right edge of a full-width line, the next line's leading
// URL-safe run is appended.
func extractURLsFromScreen(text string) []string {
	lines := strings.Split(text, "\n")
	var urls []string
	for i, line := range lines {
		for _, match := range urlRe.FindAllString(line, -1) {
			url := match
			segment := match // portion of the URL on the current line
			j := i
			current := strings.TrimRight(line, " ")
			for strings.HasSuffix(current, segment) &&
				len(current) >= wrapWidthThreshold &&
				j+1 < len(lines) {
				next := strings.TrimLeft(lines[j+1], " \t│")
				cont := urlContinuationRe.FindString(next)
				if cont == "" {
					break
				}
				url += cont
				segment = cont
				j++
				current = strings.TrimRight(lines[j], " ")
			}
			if u := trimURL(url); u != "" {
				urls = append(urls, u)
			}
		}
	}
	return urls
}

type Link struct {
	Url    string `json:"url" doc:"The extracted URL"`
	Source string `json:"source" doc:"Where the URL was found: 'timeline' (exact, from the agent's transcript) or 'message' (screen-scraped, de-wrapped heuristically)"`
	Id     int    `json:"id" doc:"Id of the timeline event or message the URL was found in"`
}

type LinksResponse struct {
	Body struct {
		Links []Link `json:"links" nullable:"false" doc:"Unique URLs in order of first appearance. Timeline-sourced URLs are preferred over screen-scraped ones."`
	}
}

// getLinks handles GET /links.
func (s *Server) getLinks(ctx context.Context, input *struct{}) (*LinksResponse, error) {
	seen := map[string]struct{}{}
	links := make([]Link, 0, 16)
	add := func(url, source string, id int) {
		if _, ok := seen[url]; ok {
			return
		}
		seen[url] = struct{}{}
		links = append(links, Link{Url: url, Source: source, Id: id})
	}

	// Timeline first: transcript content is never line-wrapped, so these
	// URLs are exact.
	for _, ev := range s.emitter.Timeline(-1, "") {
		for _, url := range extractURLs(ev.Content) {
			add(url, "timeline", ev.Id)
		}
		if len(ev.ToolInput) > 0 {
			for _, url := range extractURLs(string(ev.ToolInput)) {
				add(url, "timeline", ev.Id)
			}
		}
	}

	// Screen-scraped messages: de-wrap heuristic. Fills the gap when the
	// timeline is disabled or the agent type is unsupported.
	s.mu.RLock()
	messages := s.conversation.Messages()
	s.mu.RUnlock()
	for _, msg := range messages {
		for _, url := range extractURLsFromScreen(msg.Message) {
			add(url, "message", msg.Id)
		}
	}

	resp := &LinksResponse{}
	resp.Body.Links = links
	return resp, nil
}
