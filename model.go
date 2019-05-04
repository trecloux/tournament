package main

import (
	"database/sql"
	"time"
)

type tournament struct {
	ID              string
	Name            string
	pointsPerWin    float64
	pointsPerDraw   float64
	pointsPerDefeat float64
	pointsPerGoal   float64
}

type match struct {
	ID               int
	PoolIndex        int
	ScheduledAt      time.Time
	HomeTeamName     string
	HomeTeamID       int
	VisitorTeamName  string
	VisitorTeamID    int
	HomeTeamGoals    sql.NullInt64
	VisitorTeamGoals sql.NullInt64
}

type pool struct {
	TournamentID string
	Index        int
}

type team struct {
	ID        int
	Name      string
	PoolIndex int
}

type teamRanking struct {
	Name        string
	Played      int
	Wins        int
	Draws       int
	Defeats     int
	Goals       int
	GoalBalance int
	Points      float64
	Rank        int
}
