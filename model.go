package main

import (
	"database/sql"
	"time"
)

type tournament struct {
	ID              string
	Name            string
	pointsPerWin    float32
	pointsPerDraw   float32
	pointsPerDefeat float32
	pointsPerGoal   float32
}

type fixture struct {
	ID               int
	ScheduledAt      time.Time
	HomeTeamName     string
	HomeTeamID       int
	VisitorTeamName  string
	VisitorTeamID    int
	HomeTeamGoals    sql.NullInt64
	VisitorTeamGoals sql.NullInt64
}

type team struct {
	ID   int
	Name string
}

type teamRanking struct {
	Name        string
	Wins        int
	Draws       int
	Defeats     int
	Goals       int
	GoalBalance int
	Points      float32
	Rank        int
}
