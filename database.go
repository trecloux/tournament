package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	migrate "github.com/rubenv/sql-migrate"
)

func initDB() *sql.DB {
	db, err := sql.Open("sqlite3", "tournament.db")
	if err != nil {
		panic(err)
	}
	if db == nil {
		panic("db nil")
	}
	migrateDB(db)
	return db
}

func migrateDB(db *sql.DB) {
	migrations := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			&migrate.Migration{
				Id: "1",
				Up: []string{
					`	
					CREATE TABLE tournament (
						id TEXT PRIMARY KEY,
						name TEXT NOT NULL,
						time_zone TEXT NOT NULL,
						points_per_win REAL NOT NULL,
						points_per_draw REAL NOT NULL,
						points_per_defeat REAL NOT NULL,
						points_per_goal REAL NOT NULL
					);
					
					CREATE TABLE team (
						id INTEGER PRIMARY KEY,
						tournament_id TEXT NOT NULL REFERENCES tournament (id),
						club_id INTEGER NOT NULL REFERENCES club (id),
						name TEXT NOT NULL
					);
					
					CREATE TABLE fixture (
						id INTEGER PRIMARY KEY,
						tournament_id TEXT NOT NULL,
						scheduled_at INTEGER NOT NULL,
						home_team_id INTEGER NOT NULL REFERENCES team (id),
						away_team_id INTEGER NOT NULL REFERENCES team (id),
						home_team_goals INTEGER,
						away_team_goals INTEGER
					);
				`},
			},
		},
	}
	n, err := migrate.Exec(db, "sqlite3", migrations, migrate.Up)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Applied %d migrations!\n", n)
}

func selectFixtures(db *sql.DB, tournamentID string) []fixture {
	sql := `
		SELECT fixture.scheduled_at, tournament.time_zone, home_team.name, away_team.name, fixture.home_team_goals, fixture.away_team_goals
		FROM fixture 
		JOIN team home_team ON fixture.home_team_id = home_team.id
		JOIN team away_team ON fixture.away_team_id = away_team.id
		JOIN tournament ON fixture.tournament_id = tournament.id
		WHERE fixture.tournament_id = $1	
		ORDER BY scheduled_at
	`
	rows, err := db.Query(sql, tournamentID)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	slice := make([]fixture, 0)
	for rows.Next() {
		fixture := fixture{}
		var scheduledAtStr string
		var timeZone string
		err2 := rows.Scan(&scheduledAtStr, &timeZone, &fixture.HomeTeamName, &fixture.AwayTeamName, &fixture.HomeTeamGoals, &fixture.AwayTeamGoals)
		if err2 != nil {
			panic(err2)
		}
		fixture.ScheduledAt = parseTimeWithLocation(scheduledAtStr, timeZone)
		slice = append(slice, fixture)
	}
	return slice
}

func selectTournamentRanking(db *sql.DB, tournamentID string) []teamRanking {
	sql := `
		WITH finished_games AS (
			SELECT *
			FROM fixture 
			WHERE tournament_id =$1 AND home_team_goals IS NOT NULL AND away_team_goals IS NOT NULL
		), team_fixtures AS (
			SELECT team.id AS id, team.name AS name, home_team_goals AS team_goals, away_team_goals AS opponent_goals
			FROM team 
			JOIN finished_games ON finished_games.home_team_id = team.id 
			WHERE team.tournament_id =$1
			UNION
			SELECT team.id AS id, team.name AS name, away_team_goals AS team_goals, home_team_goals AS opponent_goals
			FROM team 
			JOIN finished_games ON finished_games.away_team_id = team.id 
			WHERE team.tournament_id =$1
		), team_result AS (
			SELECT 
				id, name,
				CASE 
					WHEN team_goals > opponent_goals THEN 1
					ELSE 0
				END AS win, 
				CASE 
					WHEN team_goals = opponent_goals THEN 1
					ELSE 0
				END AS draw, 
				CASE 
					WHEN team_goals < opponent_goals THEN 1
					ELSE 0
				END AS defeat,
				team_goals,
				opponent_goals
			FROM team_fixtures
		), team_summary AS (
			SELECT team_result.id, team_result.name, SUM(win) AS win_count, SUM(draw) AS draw_count , SUM(defeat) AS defeat_count, SUM(team_goals) AS goals, (SUM(team_goals) - SUM(opponent_goals)) AS goal_balance,
				(SUM(win)*points_per_win)
				+
				(SUM(draw)*points_per_draw)
				+
				(SUM(defeat)*points_per_defeat)
				+
				(SUM(team_goals)*points_per_goal) AS points
			FROM team_result
			JOIN tournament 
			WHERE tournament.id = $1
			GROUP BY team_result.id, team_result.name
		)
		SELECT name, win_count, draw_count, defeat_count, goals, goal_balance, points, RANK () OVER ( ORDER BY points DESC, goal_balance DESC ) rank
		FROM team_summary
		ORDER BY points DESC, goal_balance DESC	
	`
	rows, err := db.Query(sql, tournamentID)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	slice := make([]teamRanking, 0)
	for rows.Next() {
		row := teamRanking{}
		err2 := rows.Scan(&row.Name, &row.Wins, &row.Draws, &row.Defeats, &row.Goals, &row.GoalBalance, &row.Points, &row.Rank)
		if err2 != nil {
			panic(err2)
		}
		slice = append(slice, row)
	}
	return slice
}

func selectTournament(db *sql.DB, tournamentID string) tournament {
	sql := `
		SELECT name
		FROM tournament
		WHERE id = $1
	`
	row := db.QueryRow(sql, tournamentID)
	tournament := tournament{}
	err2 := row.Scan(&tournament.Name)
	if err2 != nil {
		panic(err2)
	}
	return tournament
}

func parseTimeWithLocation(timeStr string, locationStr string) time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05", timeStr)
	log.Println(t.Format(time.RFC3339))
	log.Println(t.Year())
	log.Println(locationStr)
	location, _ := time.LoadLocation(locationStr)
	y, m, d := t.Date()
	log.Println(y)
	H, M, S := t.Clock()
	t = time.Date(y, m, d, H, M, S, t.Nanosecond(), location)
	log.Println(t.Format(time.RFC3339))
	return t
}
