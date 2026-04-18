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
	DisplayOrder       int
	Type               string
}

func FromGetGameInfoAndUserProgressAchievement(achievement models.GetGameInfoAndUserProgressAchievement) Achievement {
	return Achievement{
		ID:                 achievement.ID,
		BadgeName:          achievement.BadgeName,
		DateEarnedHardcore: achievement.DateEarnedHardcore,
		Title:              achievement.Title,
		Description:        achievement.Description,
		Points:             achievement.Points,
		DisplayOrder:       achievement.DisplayOrder,
		Type:               achievement.Type,
	}
}

func FromGetAchievementOfTheWeekAchievement(achievement models.GetAchievementOfTheWeekAchievement) Achievement {
	var achievementType string
	if achievement.Type != nil {
		achievementType = *achievement.Type
	}
	return Achievement{
		ID:                 achievement.ID,
		BadgeName:          achievement.BadgeName,
		DateEarnedHardcore: nil,
		Title:              achievement.Title,
		Description:        achievement.Description,
		Points:             achievement.Points,
		Type:               achievementType,
	}
}
