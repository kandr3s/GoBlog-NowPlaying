package nowplaying

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/carlmjohnson/requests"
	"go.goblog.app/app/pkgs/bufferpool"
	"go.goblog.app/app/pkgs/htmlbuilder"
	"go.goblog.app/app/pkgs/plugintypes"
)

type plugin struct {
	app plugintypes.App

	apiKey string
	user   string

	nowPlaying *Track
}

func GetPlugin() (plugintypes.SetConfig, plugintypes.SetApp, plugintypes.UI) {
	p := &plugin{}
	return p, p, p
}

type Lfm struct {
	Recenttracks *Recenttracks `xml:"recenttracks"`
}

type Recenttracks struct {
	Track []*Track `xml:"track"`
}

type Track struct {
	Nowplaying string `xml:"nowplaying,attr"`
	Artist     *struct {
		Text string `xml:",chardata"`
	} `xml:"artist"`
	Name  string `xml:"name"`
	Album *struct {
		Text string `xml:",chardata"`
	} `xml:"album"`
	URL   string `xml:"url"`
	Image []*struct {
		Text string `xml:",chardata"`
		Size string `xml:"size,attr"`
	} `xml:"image"`
}

func (p *plugin) SetConfig(config map[string]any) {
	if lastfmAPI, ok := config["key"]; ok {
		if configlastfmKey, ok := lastfmAPI.(string); ok {
			p.apiKey = configlastfmKey
		} else {
			fmt.Println("No Last.FM API provided.")
		}
	}
	if lastfmUser, ok := config["user"]; ok {
		if configlastfmUser, ok := lastfmUser.(string); ok {
			p.user = configlastfmUser
		} else {
			fmt.Println("No Last.FM user provided.")
		}
	}
}

func (p *plugin) SetApp(app plugintypes.App) {
	p.app = app

	// Start ticker to refresh now playing every 5 minutes
	ticker := time.NewTicker(2 * time.Minute)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				fmt.Println("nowplaying plugin: Fetch now playing at", t)
				p.fetchNowPlaying()
			}
		}
	}()

	// Run once
	p.fetchNowPlaying()
}

func (p *plugin) fetchNowPlaying() {
	// Check config
	if p == nil || p.apiKey == "" || p.user == "" {
		fmt.Println("nowplaying plugin: Not configured")
		return
	}
	// Remember previous playing
	hadPrevious := p.nowPlaying != nil
	previousUrl := ""
	if hadPrevious {
		previousUrl = p.nowPlaying.URL
	}
	// Create exit function that clears now playing and cache on errors
	exit := func() {
		p.nowPlaying = nil
		if hadPrevious {
			p.app.PurgeCache()
		}
	}
	// Fetch current now playing
	result := &Lfm{}
	pr, pw := io.Pipe()
	go func() {
		_ = pw.CloseWithError(
			requests.URL("http://ws.audioscrobbler.com/2.0/").
				Param("method", "user.getrecenttracks").
				Param("limit", "3").
				Param("user", p.user).
				Param("api_key", p.apiKey).
				Client(p.app.GetHTTPClient()).
				ToWriter(pw).
				Fetch(context.Background()),
		)
	}()
	err := xml.NewDecoder(pr).Decode(result)
	_ = pr.CloseWithError(err)
	if err != nil {
		exit()
		return
	}
	// Check result
	recents := result.Recenttracks
	if recents == nil {
		exit()
		return
	}
	tracks := recents.Track
	if tracks == nil {
		exit()
		return
	}
	p.nowPlaying = nil
	for _, track := range tracks {
		if track.Nowplaying != "true" {
			continue
		}
		if track.URL != previousUrl {
			p.nowPlaying = track
			p.app.PurgeCache()
		}
		return
	}
	// Clear GoBlog cache
	if hadPrevious {
		p.app.PurgeCache()
	}
}

func (p *plugin) Render(rc plugintypes.RenderContext, rendered io.Reader, modified io.Writer) {
	if p.nowPlaying == nil {
		_, _ = io.Copy(modified, rendered)
		return
	}

	doc, err := goquery.NewDocumentFromReader(rendered)
	if err != nil {
		return
	}

	buf := bufferpool.Get()
	defer bufferpool.Put(buf)
	hb := htmlbuilder.NewHtmlBuilder(buf)

	track := p.nowPlaying

	hb.WriteElementOpen("p", "id", "nowplaying")
	hb.WriteEscaped("🎶 ")
	hb.WriteElementOpen("strong")
	hb.WriteEscaped(track.Name)
	hb.WriteElementClose("strong")
	hb.WriteEscaped(" by ")
	hb.WriteElementOpen("strong")
	hb.WriteEscaped(track.Artist.Text)
	hb.WriteElementClose("strong")
	hb.WriteElementClose("p")

	doc.Find("body header").AppendHtml(buf.String())
	_ = goquery.Render(modified, doc.Selection)
}
