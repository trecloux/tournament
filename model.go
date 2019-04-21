package main

import (
	"database/sql"
	"time"
)

type tournament struct {
	Name string
}

type fixture struct {
	ScheduledAt   time.Time
	HomeTeamName  string
	AwayTeamName  string
	HomeTeamGoals sql.NullInt64
	AwayTeamGoals sql.NullInt64
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
