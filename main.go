package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/ebitengine/gomobile/geom"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/joshraphael/go-retroachievements"
	"github.com/joshraphael/go-retroachievements/models"
	"github.com/mitchellh/go-wordwrap"
	"gra/font"
	"gra/icon"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"net/http"
	"slices"
	"strings"
	"time"
)

const (
	BaseBadgeUrl     = "https://media.retroachievements.org/Badge/"
	DefaultBadgeSize = 64
	Spacer           = 64

	ModeAuto   Mode = "Auto"
	ModeManual Mode = "Manual"
)

type Mode = string

var (
	ImageCache = make(map[string]*ebiten.Image)

	//go:embed border.png
	BorderImage []byte

	//go:embed trophy.png
	TrophyImage []byte
)

type Gra struct {
	SelectedAchievement int
	UserProgress        *models.GetGameInfoAndUserProgress
	OrderedAchievements []models.GetGameInfoAndUserProgressAchievement
	BorderImage         *image.Image
	TrophyImage         *image.Image
	Client              *retroachievements.Client
	Config              *Config
	LatestRefresh       time.Time
	FontSource          *text.GoTextFaceSource
	CurrentMode         Mode
}

func (g *Gra) Update() error {
	var nextRefresh = g.LatestRefresh.Add(g.Config.Connect.RefreshInterval * time.Second)
	if time.Now().After(nextRefresh) {
		err := g.refreshAchievements()
		if err != nil {
			return fmt.Errorf("error refreshing achievements: %w", err)
		}
		g.LatestRefresh = time.Now()
	}

	//Auto mode
	if g.CurrentMode == ModeAuto {
		firstUnbeaten := 0
		for i, achievement := range g.OrderedAchievements {
			if achievement.DateEarnedHardcore == nil {
				firstUnbeaten = i
				break
			}
		}
		g.SelectedAchievement = firstUnbeaten
	}

	g.handleInput()

	return nil
}

func (g *Gra) Draw(screen *ebiten.Image) {
	g.drawAchievements(screen)
	g.drawCurrentAchievement(screen)
}

func (g *Gra) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	if g.UserProgress != nil {
		rows := g.getNumberOfAchievementRows()
		size := g.getAchievementSize()
		details := Spacer + 200
		rowSize := g.Config.Display.AchievementsPerRow * size
		if rows*size+details != outsideHeight || rowSize != outsideWidth {
			ebiten.SetWindowSize(rowSize, rows*size+details)
		}
	}
	return outsideWidth, outsideHeight
}

func (g *Gra) getNumberOfAchievementRows() int {
	return int(math.Ceil(float64(g.UserProgress.NumAchievements) / float64(g.Config.Display.AchievementsPerRow)))
}

func (g *Gra) getAchievementSize() int {
	return DefaultBadgeSize
}

func (g *Gra) refreshAchievements() error {
	recent, err := g.Client.GetUserRecentlyPlayedGames(models.GetUserRecentlyPlayedGamesParameters{
		Username: g.Config.Connect.Username,
	})
	if err != nil {
		return err
	}

	if len(recent) == 0 {
		return fmt.Errorf("no recent games found")
	}

	progress, err := g.Client.GetGameInfoAndUserProgress(models.GetGameInfoAndUserProgressParameters{
		Username: g.Config.Connect.Username,
		GameID:   recent[0].GameID,
	})
	if err != nil {
		return err
	}

	var orderedAchievements []models.GetGameInfoAndUserProgressAchievement
	for _, achievement := range progress.Achievements {
		orderedAchievements = append(orderedAchievements, achievement)
	}
	slices.SortFunc(orderedAchievements, func(a, b models.GetGameInfoAndUserProgressAchievement) int {
		return a.ID - b.ID
	})

	g.UserProgress = progress
	g.OrderedAchievements = orderedAchievements
	return nil
}

func (g *Gra) DrawTitle(screen *ebiten.Image) {
	var title string
	if g.UserProgress == nil {
		title = "Loading..."
	} else {
		title = g.UserProgress.Title
	}
	op := &text.DrawOptions{}
	op.PrimaryAlign = text.AlignCenter
	op.GeoM.Translate(float64(screen.Bounds().Dx()/2), 10)
	op.ColorScale.ScaleWithColor(color.White)

	text.Draw(screen, title, &text.GoTextFace{
		Source: g.FontSource,
		Size:   16,
	}, op)
}

func (g *Gra) drawAchievements(screen *ebiten.Image) {
	initialOffsets := geom.Point{X: 0, Y: 0}
	achievementSize := float64(DefaultBadgeSize)

	if g.UserProgress == nil {
		return
	}

	var currentRow float64 = 0
	for i, achievement := range g.OrderedAchievements {
		geo := ebiten.GeoM{}
		geo.Translate(float64(initialOffsets.X), float64(initialOffsets.Y))

		if i%g.Config.Display.AchievementsPerRow == 0 && i != 0 {
			currentRow++
		}

		geo.Translate(achievementSize*float64(i%g.Config.Display.AchievementsPerRow), 64*currentRow)

		badge, err := loadBadge(achievement.BadgeName, achievement.DateEarnedHardcore != nil)
		if err != nil {
			log.Printf("error loading badge: %v", err)
			return
		}
		screen.DrawImage(ebiten.NewImageFromImage(badge), &ebiten.DrawImageOptions{
			GeoM: geo,
		})

		if i == g.SelectedAchievement {
			screen.DrawImage(ebiten.NewImageFromImage(*g.BorderImage), &ebiten.DrawImageOptions{
				GeoM: geo,
			})
		}
	}

}

func (g *Gra) handleInput() {
	if g.UserProgress == nil {
		return
	}

	if !g.Config.Display.DisableAutoMode {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.CurrentMode = ModeAuto
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) {
		g.Config.Display.AchievementsPerRow++
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) {
		if g.Config.Display.AchievementsPerRow > 1 {
			g.Config.Display.AchievementsPerRow--
		}

	}
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.SelectedAchievement -= 1
		if g.SelectedAchievement < 0 {
			g.SelectedAchievement = 0
		}
		g.CurrentMode = ModeManual
	} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.SelectedAchievement += 1
		g.CurrentMode = ModeManual
	} else if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.SelectedAchievement -= g.Config.Display.AchievementsPerRow
		if g.SelectedAchievement < 0 {
			g.SelectedAchievement = 0
		}
		g.CurrentMode = ModeManual
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.SelectedAchievement += g.Config.Display.AchievementsPerRow
		g.CurrentMode = ModeManual
	}

	if g.SelectedAchievement >= g.UserProgress.NumAchievements {
		g.SelectedAchievement = g.UserProgress.NumAchievements - 1
	}
}

func (g *Gra) drawCurrentAchievement(screen *ebiten.Image) {
	if g.UserProgress == nil {
		return
	}

	initialOffsets := geom.Point{X: 10, Y: geom.Pt(g.getNumberOfAchievementRows()*g.getAchievementSize() + Spacer)}

	geo := ebiten.GeoM{}
	geo.Translate(float64(initialOffsets.X), float64(initialOffsets.Y))

	achievement := g.OrderedAchievements[g.SelectedAchievement]
	badge, err := loadBadge(achievement.BadgeName, achievement.DateEarnedHardcore != nil)
	if err != nil {
		return
	}
	screen.DrawImage(ebiten.NewImageFromImage(badge), &ebiten.DrawImageOptions{
		GeoM: geo,
	})

	if achievement.DateEarnedHardcore != nil {
		geoT := ebiten.GeoM{}
		geoT.Translate(float64(initialOffsets.X/2), float64(initialOffsets.Y/2))
		geoT.Translate(0, Spacer/2+10)
		geoT.Scale(2, 2)

		screen.DrawImage(ebiten.NewImageFromImage(ebiten.NewImageFromImage(*g.TrophyImage)), &ebiten.DrawImageOptions{
			GeoM: geoT,
		})
		g.drawText(screen, float64(initialOffsets.X+32+3), float64(initialOffsets.Y+DefaultBadgeSize+DefaultBadgeSize+24), "Done!", text.AlignCenter, color.RGBA{G: 255})
	}

	//Achievement details
	g.drawText(screen, float64(screen.Bounds().Dx()/2), float64(initialOffsets.Y-5-42), "[Selected Achievement]", text.AlignCenter, color.White)
	g.drawText(screen, float64(initialOffsets.X+DefaultBadgeSize+20), float64(initialOffsets.Y-5), achievement.Title, text.AlignStart, color.White)
	g.drawText(screen, float64(initialOffsets.X+DefaultBadgeSize+20), float64(initialOffsets.Y+27), achievement.Description, text.AlignStart, color.White)

	//Mode
	if !g.Config.Display.HideMode { // No need to draw mode if manual is forced
		g.drawText(screen, float64(screen.Bounds().Max.X), float64(screen.Bounds().Max.Y-26), g.CurrentMode, text.AlignEnd, color.Gray{Y: 100})
	}
}

func (g *Gra) drawText(screen *ebiten.Image, x float64, y float64, txt string, align text.Align, color color.Color) {
	txt = wordwrap.WrapString(txt, 36)
	op := &text.DrawOptions{}
	op.PrimaryAlign = align
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(color)
	for i, line := range strings.Split(txt, "\n") {
		line = strings.TrimSpace(line)
		if i != 0 {
			op.GeoM.Translate(0, 24)
		}
		text.Draw(screen, line, &text.GoTextFace{
			Source: g.FontSource,
			Size:   32,
		}, op)
	}
}

func loadBadge(name string, earned bool) (*ebiten.Image, error) {
	var fullName string
	if earned {
		fullName = name + ".png"
	} else {
		fullName = name + "_lock.png"
	}

	badge, ok := ImageCache[fullName]
	if !ok {
		resp, err := http.Get(BaseBadgeUrl + fullName)
		if err != nil {
			return nil, err
		}
		defer panicIfError(resp.Body.Close)
		rawPNG, err := png.Decode(resp.Body)
		if err != nil {
			return nil, err
		}
		badge = ebiten.NewImageFromImage(rawPNG)
		ImageCache[fullName] = badge
	}

	return badge, nil
}

func panicIfError(f func() error) {
	if err := f(); err != nil {
		panic(err)
	}
}

type Config struct {
	Connect struct {
		Username        string        `toml:"username"`
		ApiKey          string        `toml:"apiKey"`
		RefreshInterval time.Duration `toml:"refreshInterval"`
	} `toml:"connect"`
	Display struct {
		AchievementsPerRow int  `toml:"achievementsPerRow"`
		DisableAutoMode    bool `toml:"disableAutoMode"`
		HideMode           bool `toml:"hideMode"`
	} `toml:"display"`
}

func main() {
	var err error

	border, trophy, err := loadImages()
	if err != nil {
		log.Fatal(err)
	}

	icons, err := icon.LoadIcons()
	if err != nil {
		log.Fatal(err)
	}

	fontSource, err := loadFont()
	if err != nil {
		log.Fatal(err)
	}

	config, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	if config.Connect.ApiKey == "" || config.Connect.Username == "" {
		log.Fatal("Connect API key or username missing")
	}

	mode := ModeAuto
	if config.Display.DisableAutoMode {
		config.Display.HideMode = true
		mode = ModeManual
	}

	gra := &Gra{
		BorderImage: border,
		TrophyImage: trophy,
		Client:      retroachievements.NewClient(config.Connect.ApiKey),
		Config:      config,
		FontSource:  fontSource,
		CurrentMode: mode,
	}

	err = gra.refreshAchievements() // Initial preload
	if err != nil {
		log.Fatal(err)
	}

	gra.Layout(0, 0)
	ebiten.SetWindowTitle("Retro Achievements")
	ebiten.SetTPS(30)
	ebiten.SetWindowIcon(icons)

	if err := ebiten.RunGame(gra); err != nil {
		log.Fatal(err)
	}
}

func loadFont() (*text.GoTextFaceSource, error) {
	return text.NewGoTextFaceSource(bytes.NewReader(font.Bookxel))
}

func loadConfig() (*Config, error) {
	var config Config
	_, err := toml.DecodeFile("config.toml", &config)
	if err != nil {
		return nil, fmt.Errorf("error loading config.toml: %s", err)
	}
	if config.Connect.RefreshInterval == 0 {
		config.Connect.RefreshInterval = 5
	}
	if config.Display.AchievementsPerRow == 0 {
		config.Display.AchievementsPerRow = 8
	}
	return &config, nil
}

func loadImages() (*image.Image, *image.Image, error) {
	var err error

	borderPNG, err := png.Decode(bytes.NewReader(BorderImage))
	if err != nil {
		return nil, nil, fmt.Errorf("error decoding border.png: %w", err)
	}

	trophyPNG, err := png.Decode(bytes.NewReader(TrophyImage))
	if err != nil {
		return nil, nil, fmt.Errorf("error decoding trophy.png: %w", err)
	}
	return &borderPNG, &trophyPNG, nil
}
