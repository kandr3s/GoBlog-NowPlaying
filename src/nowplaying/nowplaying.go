package nowplaying

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"strconv"

	"github.com/PuerkitoBio/goquery"
	"go.goblog.app/app/pkgs/bufferpool"
	"go.goblog.app/app/pkgs/htmlbuilder"
	"go.goblog.app/app/pkgs/plugintypes"
)

type plugin struct {
	app           plugintypes.App
	fmkey string
	fmuser string
}

type NowPlaying struct {
	Recenttracks struct {
		Track []struct {
			Artist struct {
				Mbid string `json:"mbid"`
				Name string `json:"#text"`
			} `json:"artist"`
			Streamable string `json:"streamable"`
			Image      []struct {
				Size string `json:"size"`
				Text string `json:"#text"`
			} `json:"image"`
			Mbid  string `json:"mbid"`
			Album struct {
				Mbid string `json:"mbid"`
				Text string `json:"#text"`
			} `json:"album"`
			Name string `json:"name"`
			Attr struct {
				Nowplaying bool `json:"nowplaying"`
			} `json:"@attr,omitempty"`
			URL  string `json:"url"`
			Date struct {
				Uts  string  `json:"uts"`
				Text string `json:"#text"`
			} `json:"date,omitempty"`
		} `json:"track"`
		Attr struct {
			User       string `json:"user"`
			TotalPages string `json:"totalPages"`
			Page       string `json:"page"`
			PerPage    string `json:"perPage"`
			Total      string `json:"total"`
		} `json:"@attr"`
	} `json:"recenttracks"`
}

func GetPlugin() (plugintypes.SetConfig, plugintypes.SetApp, plugintypes.UI) {
	p := &plugin{}
	return p, p, p
}

func (p *plugin) SetConfig(config map[string]any) {
	if lastfmAPI, ok := config["fmkey"]; ok {
		if configlastfmKey, ok := lastfmAPI.(string); ok {
			p.fmkey = configlastfmKey
			if lastfmUser, ok := config["fmuser"]; ok {
				if configlastfmUser, ok := lastfmUser.(string); ok {
					p.fmuser = configlastfmUser 
				}  else {
					fmt.Println("No Last.FM user provided.")
				}	
			}
		} else {
			fmt.Println("No Last.FM API provided.")
		}
	} 
}

func (p *plugin) SetApp(app plugintypes.App) {
	p.app = app
}

func (p *plugin) Render(rc plugintypes.RenderContext, rendered io.Reader, modified io.Writer) {
	blog := rc.GetBlog()
	if blog == "" {
		fmt.Println("nowplaying plugin: blog is empty!")
		return
	}
	doc, err := goquery.NewDocumentFromReader(rendered)
	if err != nil {
		fmt.Println("webrings plugin: " + err.Error())
		return
	}
	result := &NowPlaying{}
	apiurl := fmt.Sprintf("http://ws.audioscrobbler.com/2.0/?method=user.getrecenttracks&user=%s&api_key=%s&limit=1&format=json", p.fmuser, p.fmkey)
	getJson(apiurl, result)
	buf := bufferpool.Get()
	defer bufferpool.Put(buf)
	hb := htmlbuilder.NewHtmlBuilder(buf)
	buf.Reset()
	
	hb.WriteElementOpen("div", "class", "nowplaying")
	for _, rec := range result.Recenttracks.Track {
		apitimestamp := rec.Date.Uts
  		unixTimestamp, err := strconv.ParseInt(apitimestamp, 10, 64)
		if err != nil {
		fmt.Println("An error occurred:", err)
		}
		timestamp := time.Unix(unixTimestamp, 0)
		elapsed :=  time.Since(timestamp)
		if elapsed <= 10 * time.Minute {
			hb.WriteElementOpen("img", "src", "https://kandr3s.co/smilies/listening.gif", "title", "Now playing", "style", "width:auto;height:25px")
			hb.WriteElementOpen("span")
			// rec.Image[3]
			hb.WriteElementOpen("marquee", "bahaviour", "scroll", "direction", "left")
			hb.WriteElementOpen("img", "src", rec.Image[0].Text, "title", rec.Name, "class", "nowplaying-art", "alt", rec.Album.Text)
			hb.WriteElementOpen("em")
			hb.WriteEscaped(rec.Name)
			hb.WriteElementClose("em")
			hb.WriteEscaped(" by ")
			hb.WriteElementOpen("b")
			hb.WriteEscaped(rec.Artist.Name)
			hb.WriteElementClose("b")
			hb.WriteElementClose("marquee")
			hb.WriteElementClose("span")
			break
		} else {
			break
		}
	}
	hb.WriteElementClose("div")
	doc.Find("main").PrependHtml(buf.String())
	_ = goquery.Render(modified, doc.Selection)
}

func getJson(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}
