package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	migrate "github.com/rubenv/sql-migrate"
)

const timeFormat = "15:04"

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
						points_per_win REAL NOT NULL,
						points_per_draw REAL NOT NULL,
						points_per_defeat REAL NOT NULL,
						points_per_goal REAL NOT NULL
					);
					
					CREATE TABLE team (
						id INTEGER PRIMARY KEY,
						tournament_id TEXT NOT NULL REFERENCES tournament (id),
						name TEXT NOT NULL
					);
					
					CREATE TABLE fixture (
						id INTEGER PRIMARY KEY,
						tournament_id TEXT NOT NULL,
						scheduled_at INTEGER NOT NULL,
						home_team_id INTEGER NOT NULL REFERENCES team (id),
						visitor_team_id INTEGER NOT NULL REFERENCES team (id),
						home_team_goals INTEGER,
						visitor_team_goals INTEGER,
						UNIQUE(tournament_id, home_team_id, visitor_team_id)						
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
		SELECT fixture.id, fixture.scheduled_at, home_team.name, visitor_team.name, fixture.home_team_goals, fixture.visitor_team_goals
		FROM fixture 
		JOIN team home_team ON fixture.home_team_id = home_team.id
		JOIN team visitor_team ON fixture.visitor_team_id = visitor_team.id
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
		err2 := rows.Scan(&fixture.ID, &scheduledAtStr, &fixture.HomeTeamName, &fixture.VisitorTeamName, &fixture.HomeTeamGoals, &fixture.VisitorTeamGoals)
		if err2 != nil {
			panic(err2)
		}
		fixture.ScheduledAt = parseTime(scheduledAtStr)
		slice = append(slice, fixture)
	}
	return slice
}

func selectTournamentRanking(db *sql.DB, tournamentID string) []teamRanking {
	sql := `
		WITH finished_games AS (
			SELECT *
			FROM fixture 
			WHERE tournament_id =$1 AND home_team_goals IS NOT NULL AND visitor_team_goals IS NOT NULL
		), team_fixtures AS (
			SELECT team.id AS id, team.name AS name, home_team_goals AS team_goals, visitor_team_goals AS opponent_goals
			FROM team 
			JOIN finished_games ON finished_games.home_team_id = team.id 
			WHERE team.tournament_id =$1
			UNION
			SELECT team.id AS id, team.name AS name, visitor_team_goals AS team_goals, home_team_goals AS opponent_goals
			FROM team 
			JOIN finished_games ON finished_games.visitor_team_id = team.id 
			WHERE team.tournament_id =$1
		), team_result AS (
			SELECT 
				id,
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
			SELECT team_result.id, SUM(win) AS win_count, SUM(draw) AS draw_count , SUM(defeat) AS defeat_count, SUM(team_goals) AS goals, (SUM(team_goals) - SUM(opponent_goals)) AS goal_balance,
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
			GROUP BY team_result.id
		)
		SELECT 
			name, 
			COALESCE(win_count, 0),
			COALESCE(draw_count, 0),
			COALESCE(defeat_count, 0),
			COALESCE(goals, 0),
			COALESCE(goal_balance, 0),
			COALESCE(points, 0),
			RANK () OVER ( ORDER BY points DESC, goal_balance DESC, name ) rank
		FROM team 
		LEFT JOIN team_summary ON team.id = team_summary.id
		ORDER BY rank	
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
		SELECT id, name
		FROM tournament
		WHERE id = $1
	`
	row := db.QueryRow(sql, tournamentID)
	tournament := tournament{}
	err2 := row.Scan(&tournament.ID, &tournament.Name)
	if err2 != nil {
		panic(err2)
	}
	return tournament
}

func selectTournamentTeams(db *sql.DB, tournamentID string) []team {
	sql := `
		SELECT team.id, team.name
		FROM team 
		JOIN tournament ON tournament.id = team.tournament_id
		WHERE tournament.id = $1
		ORDER BY team.id	
	`
	rows, err := db.Query(sql, tournamentID)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	slice := make([]team, 0)
	for rows.Next() {
		row := team{}
		err2 := rows.Scan(&row.ID, &row.Name)
		if err2 != nil {
			panic(err2)
		}
		slice = append(slice, row)
	}
	return slice
}

func selectTournaments(db *sql.DB) []tournament {
	sql := `
		SELECT id, name
		FROM tournament
		ORDER BY id	
	`
	rows, err := db.Query(sql)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	slice := make([]tournament, 0)
	for rows.Next() {
		row := tournament{}
		err2 := rows.Scan(&row.ID, &row.Name)
		if err2 != nil {
			panic(err2)
		}
		slice = append(slice, row)
	}
	return slice
}

func insertTournament(db *sql.DB, t tournament) {
	sql := `
		INSERT INTO tournament(id, name, points_per_win, points_per_draw, points_per_defeat, points_per_goal)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Exec(sql, t.ID, t.Name, t.pointsPerWin, t.pointsPerDraw, t.pointsPerDefeat, t.pointsPerGoal)
	if err != nil {
		panic(err)
	}
}
func insertTeams(db *sql.DB, tournamentID string, teams []team) {
	sql := `
		INSERT INTO team(tournament_id, name)
		VALUES ($1, $2)
	`
	for _, team := range teams {
		_, err := db.Exec(sql, tournamentID, team.Name)
		if err != nil {
			panic(err)
		}
	}
}
func insertFixtures(db *sql.DB, tournamentID string, fixtures []fixture) {
	sql := `
		INSERT INTO fixture(tournament_id, scheduled_at, home_team_id, visitor_team_id)
		VALUES ($1, $2, $3, $4)
	`
	for _, fixture := range fixtures {
		_, err := db.Exec(sql, tournamentID, fixture.ScheduledAt.Format(timeFormat), fixture.HomeTeamID, fixture.VisitorTeamID)
		if err != nil {
			panic(err)
		}
	}
}
func deleteTournament(db *sql.DB, tournamentID string) {
	sql := "DELETE FROM fixture WHERE tournament_id = $1"
	_, err := db.Exec(sql, tournamentID)
	if err != nil {
		panic(err)
	}
	sql = "DELETE FROM team WHERE tournament_id = $1"
	_, err = db.Exec(sql, tournamentID)
	if err != nil {
		panic(err)
	}
	sql = "DELETE FROM tournament WHERE id = $1"
	_, err = db.Exec(sql, tournamentID)
	if err != nil {
		panic(err)
	}
}

func saveFixtureScore(db *sql.DB, fixtureID int, homeTeamGoals int, visitorTeamGoals int) {
	sql := "UPDATE fixture SET home_team_goals=$1, visitor_team_goals=$2 WHERE id = $3"
	_, err := db.Exec(sql, homeTeamGoals, visitorTeamGoals, fixtureID)
	if err != nil {
		panic(err)
	}
}

func parseTime(timeStr string) time.Time {
	t, _ := time.Parse(timeFormat, timeStr)
	return t
}
