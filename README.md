## GoBlog NowPlaying Plugin

Plugin for GoBlog blogging system that displays the currently playing song from Last.FM.

## Installation

1.  Copy the “nowplaying” folder into your plugins folder.
2.  Add the following config to your config.yml

```text-plain
- path: ./plugins/nowplaying
  import: nowplaying
  config:
      lastfmUser: yourLastFMNick
      lastfmAPI: yourLastFMAPIKey
```

Plugin can be seen in action on [my website](https://kandr3s.co) whenever a song is actually being played.
