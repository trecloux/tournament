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
	Pools           []pool
}

type poolMatch struct {
	ID               int
	PoolIndex        int
	ScheduledAt      time.Time
	HomeTeamName     string
	HomeTeamID       int
	VisitorTeamName  string
	VisitorTeamID    int
	HomeTeamGoals    sql.NullInt64
	VisitorTeamGoals sql.NullInt64
	PitchName        string
}

type rankingMatch struct {
	Key                                 string
	ScheduledAt                         time.Time
	HomeTeamPoolIndex                   sql.NullInt64
	HomeTeamPoolRank                    sql.NullInt64
	HomeTeamSourceRankingMatch          sql.NullString
	HomeTeamSourceRankingMatchWinner    sql.NullBool
	HomeTeamName                        sql.NullString
	HomeTeamGoals                       sql.NullInt64
	HomeTeamID                          sql.NullInt64
	VisitorTeamPoolIndex                sql.NullInt64
	VisitorTeamPoolRank                 sql.NullInt64
	VisitorTeamSourceRankingMatch       sql.NullString
	VisitorTeamSourceRankingMatchWinner sql.NullBool
	VisitorTeamName                     sql.NullString
	VisitorTeamGoals                    sql.NullInt64
	VisitorTeamID                       sql.NullInt64
	WinnerTeamID                        sql.NullInt64
	LooserTeamID                        sql.NullInt64
	ValidTeams                          bool
	PitchName                           string
	PenaltyShootOutWinner               string
}

type pool struct {
	TournamentID string
	Index        int
	Name         string
}

type poolViewModel struct {
	PoolIndex int
	PoolName  string
	Matches   []poolMatch
}

type rankingViewModel struct {
	PoolIndex    int
	PoolName     string
	TeamRankings []teamRanking
}

type team struct {
	ID        int
	Name      string
	PoolIndex int
}

type teamRanking struct {
	ID            int
	Name          string
	Played        int
	Wins          int
	Draws         int
	Defeats       int
	TeamGoals     int
	OpponentGoals int
	GoalBalance   int
	Points        float64
	Rank          int
}

type tournamentFinalRanking struct {
	Rank     int
	TeamName sql.NullString
}
