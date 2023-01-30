## GoBlog NowPlaying Plugin

Plugin for GoBlog blogging system that displays the currently playing song from Last.FM.

## Installation

1.  Copy the “nowplaying” folder into your plugins folder.
2.  Add the following config to your config.yml

```yaml
plugins:
  - path: ./plugins/nowplaying
    import: nowplaying
    config:
      user: yourLastFMNick
      key: yourLastFMAPIKey
```

Plugin can be seen in action on [my website](https://kandr3s.co) whenever a song is actually being played.
