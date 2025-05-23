# GRA! A Go Retro Achievement tracker
A small tracker meant to showcase the current progress of your achievement hunting,
as well as to easily navigate which ones remain.

## Screenshot
![Screenshot of tool](screenshot.png)
## Build
Check out the repo and to the following
```bash
go get
go build
```
The binary `gra` should be available in the same folder.

## Run
1. Either grab a binary from the releases page, or check out the repo and build the source.
2. Copy config.toml.example to config.toml, fill in the needed information (username and api key)
3. Run the program with `./gra`

## Config

The program will look for the config.toml file at `current folder`, as well as `~/.config/gra/config.toml`.

| Config             | Category | Type   | Required | Comment                                                                                        |
|--------------------|----------|--------|----------|------------------------------------------------------------------------------------------------|
| username           | connect  | string | yes      | RA username of user whom the application should track.                                         |
| apiKey             | connect  | string | yes      | RA Api key, used to authenticate the user, get one from https://retroachievements.org/settings |
| refreshInterval    | connect  | int    | no       | How often in seconds the application should poll for new information from RA. **Default: 5**   |
| achievementsPerRow | display  | int    | no       | How many achievements to display per row. **Default: 8**                                       |
| disableAutoMode    | display  | bool   | no       | Disable Auto Mode. **Default: false**                                                          |
| hideMode           | display  | bool   | no       | Disable Mode display in lower right corner. **Default: false**                                 |
| swapFallbackFont   | display  | bool   | no       | Use the fallback font primarily instead. **Default: false**                                    |

## Keys
| Key                      | Note                                                                 |
|--------------------------|----------------------------------------------------------------------|
| Arrow Up/Left/Right/Down | Move achievement selector, will trigger *Manual Mode*                |
| ESC                      | Switch to *Auto Mode* (or *Manual Mode* if *Auto Mode* is disabled)  |
| W                        | Show Achievement of the week, press again to return to *Manual Mode* |
| F                        | Swap to fallback font with better glyph support                      |
| Keypad +                 | Increase achievements per row by 1                                   |
| Keypad -                 | Decrease achievements per row by 1                                   |

## Libraries Used
These are the main awesome libraries used for building *GRA!* go give them a star!

| Library              | URL                                                 | Description                                               |
|----------------------|-----------------------------------------------------|-----------------------------------------------------------|
| ebiten               | https://github.com/hajimehoshi/ebiten               | Game engine, used for the ease of rendering the app in GL |
| go-retroachievements | https://github.com/joshraphael/go-retroachievements | Go library for retro achievements web API                 |