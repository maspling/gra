package main

import (
	"bytes"
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
	"os"
	"slices"
	"strings"
	"time"
)

const (
	BaseBadgeUrl     = "https://media.retroachievements.org/Badge/"
	DefaultBadgeSize = 64
	Spacer           = 64
)

var (
	ImageCache = make(map[string]*ebiten.Image)
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
	return DefaultBadgeSize * g.Config.Display.AchievementSizeMultiple
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
	achievementSize := float64(DefaultBadgeSize * g.Config.Display.AchievementSizeMultiple)

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
	if inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) {
		g.Config.Display.AchievementsPerRow++
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) {
		g.Config.Display.AchievementsPerRow--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.SelectedAchievement -= 1
		if g.SelectedAchievement < 0 {
			g.SelectedAchievement = 0
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.SelectedAchievement += 1
	} else if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.SelectedAchievement -= g.Config.Display.AchievementsPerRow
		if g.SelectedAchievement < 0 {
			g.SelectedAchievement = 0
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.SelectedAchievement += g.Config.Display.AchievementsPerRow
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
		g.drawText(screen, float64(initialOffsets.X+32+3), float64(initialOffsets.Y+64+64+24), "Done!", text.AlignCenter, color.RGBA{0, 255, 0, 0})
	}

	// Achievement Title

	g.drawText(screen, float64(initialOffsets.X+64+20), float64(initialOffsets.Y-5), achievement.Title, text.AlignStart, color.White)
	g.drawText(screen, float64(initialOffsets.X+64+20), float64(initialOffsets.Y+27), achievement.Description, text.AlignStart, color.White)
}

func (g *Gra) drawText(screen *ebiten.Image, x float64, y float64, txt string, align text.Align, color color.Color) {
	txt = wordwrap.WrapString(txt, 38)
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
		defer resp.Body.Close()
		rawPNG, err := png.Decode(resp.Body)
		if err != nil {
			return nil, err
		}
		badge = ebiten.NewImageFromImage(rawPNG)
		ImageCache[fullName] = badge
	}

	return badge, nil
}

type Config struct {
	Connect struct {
		Username        string        `toml:"username"`
		ApiKey          string        `toml:"apiKey"`
		RefreshInterval time.Duration `toml:"refreshInterval"`
	} `toml:"connect"`
	Display struct {
		AchievementsPerRow      int `toml:"achievementsPerRow"`
		AchievementSizeMultiple int `toml:"achievementSizeMultiple"`
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

	gra := &Gra{
		BorderImage: border,
		TrophyImage: trophy,
		Client:      retroachievements.NewClient(config.Connect.ApiKey),
		Config:      config,
		FontSource:  fontSource,
	}

	err = gra.refreshAchievements() // Initial preload
	if err != nil {
		log.Fatal(err)
	}

	startingWidth := DefaultBadgeSize * config.Display.AchievementsPerRow * config.Display.AchievementSizeMultiple
	ebiten.SetWindowSize(startingWidth, gra.getNumberOfAchievementRows())
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
	return &config, nil
}

func loadImages() (*image.Image, *image.Image, error) {
	var err error

	border, err := os.Open("border.png")
	if err != nil {
		return nil, nil, fmt.Errorf("error opening border.png: %w", err)
	}
	defer border.Close()

	trophy, err := os.Open("trophy.png")
	if err != nil {
		return nil, nil, fmt.Errorf("error opening trophy.png: %w", err)
	}
	defer border.Close()

	borderPNG, err := png.Decode(border)
	if err != nil {
		return nil, nil, fmt.Errorf("error decoding border.png: %w", err)
	}

	trophyPNG, err := png.Decode(trophy)
	if err != nil {
		return nil, nil, fmt.Errorf("error decoding trophy.png: %w", err)
	}
	return &borderPNG, &trophyPNG, nil
}
