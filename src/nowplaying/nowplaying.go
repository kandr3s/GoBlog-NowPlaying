package nowplaying

import (
	"context"
	"fmt"
	"io"
	"strconv"
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

type NowPlaying struct {
	Recenttracks *struct {
		Track []*Track `json:"track"`
	} `json:"recenttracks"`
}

type Track struct {
	Artist *struct {
		Mbid string `json:"mbid"`
		Text string `json:"#text"`
	} `json:"artist"`
	Streamable string `json:"streamable"`
	Image      []*struct {
		Size string `json:"size"`
		Text string `json:"#text"`
	} `json:"image"`
	Mbid  string `json:"mbid"`
	Album *struct {
		Mbid string `json:"mbid"`
		Text string `json:"#text"`
	} `json:"album"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Date *struct {
		Uts  string `json:"uts"`
		Text string `json:"#text"`
	} `json:"date"`
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
	ticker := time.NewTicker(5 * time.Minute)
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
	if p == nil || p.apiKey == "" || p.user == "" {
		fmt.Println("nowplaying plugin: Not configured")
		return
	}
	hadPrevious := p.nowPlaying != nil
	exit := func() {
		p.nowPlaying = nil
		if hadPrevious {
			p.app.PurgeCache()
		}
	}
	// Fetch current now playing
	result := &NowPlaying{}
	err := requests.URL("http://ws.audioscrobbler.com/2.0/").
		Param("method", "user.getrecenttracks").
		Param("limit", "1").
		Param("format", "json").
		Param("user", p.user).
		Param("api_key", p.apiKey).
		Client(p.app.GetHTTPClient()).
		ToJSON(result).
		Fetch(context.Background())
	if err != nil {
		exit()
		return
	}
	// Save, if played in the last 10 minutes
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
		if track.Date == nil || track.Date.Uts == "" {
			continue
		}
		unixTimestamp, _ := strconv.ParseInt(track.Date.Uts, 10, 64)
		timestamp := time.Unix(int64(unixTimestamp), 0)
		if time.Since(timestamp) < 10*time.Minute {
			p.nowPlaying = track
		}
		break
	}
	// Clear GoBlog cache
	if hadPrevious || p.nowPlaying != nil {
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
