package achievement

import (
	"github.com/joshraphael/go-retroachievements/models"
)

type Achievement struct {
	ID                 int
	BadgeName          string
	DateEarnedHardcore *models.DateTime
	Title              string
	Description        string
	Points             int
}

func FromGetGameInfoAndUserProgressAchievement(achievement models.GetGameInfoAndUserProgressAchievement) Achievement {
	return Achievement{
		ID:                 achievement.ID,
		BadgeName:          achievement.BadgeName,
		DateEarnedHardcore: achievement.DateEarnedHardcore,
		Title:              achievement.Title,
		Description:        achievement.Description,
		Points:             achievement.Points,
	}
}

func FromGetAchievementOfTheWeekAchievement(achievement models.GetAchievementOfTheWeekAchievement) Achievement {
	return Achievement{
		ID:                 achievement.ID,
		BadgeName:          achievement.BadgeName,
		DateEarnedHardcore: nil,
		Title:              achievement.Title,
		Description:        achievement.Description,
		Points:             achievement.Points,
	}
}
