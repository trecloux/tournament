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
					
					CREATE TABLE pool (
						tournament_id TEXT NOT NULL REFERENCES tournament(id),
						pool_index INTEGER NOT NULL,
						PRIMARY KEY(tournament_id, pool_index)
					);
					
					CREATE TABLE team (
						id INTEGER PRIMARY KEY,
						tournament_id TEXT NOT NULL REFERENCES tournament(id),
						pool_index INTEGER NOT NULL REFERENCES pool(pool_index),
						name TEXT NOT NULL
					);
					
					CREATE TABLE match (
						id INTEGER PRIMARY KEY,
						tournament_id TEXT NOT NULL NOT NULL REFERENCES tournament(id),
						pool_index INTEGER REFERENCES pool(pool_index),
						scheduled_at INTEGER NOT NULL,
						home_team_id INTEGER NOT NULL REFERENCES team(id),
						visitor_team_id INTEGER NOT NULL REFERENCES team(id),
						home_team_goals INTEGER,
						visitor_team_goals INTEGER,
						UNIQUE(tournament_id, pool_index, home_team_id, visitor_team_id)						
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

func selectTournamentMatches(db *sql.DB, tournamentID string) []match {
	sql := `
		SELECT match.id, match.pool_index, match.scheduled_at, home_team.name, visitor_team.name, match.home_team_goals, match.visitor_team_goals
		FROM match 
		JOIN team home_team ON match.home_team_id = home_team.id
		JOIN team visitor_team ON match.visitor_team_id = visitor_team.id
		WHERE match.tournament_id = $1
		ORDER BY scheduled_at
	`
	return fetchMatches(db.Query(sql, tournamentID))
}
func selectTournamentPoolMatches(db *sql.DB, tournamentID string, poolIndex int) []match {
	sql := `
		SELECT match.id, match.pool_index, match.scheduled_at, home_team.name, visitor_team.name, match.home_team_goals, match.visitor_team_goals
		FROM match 
		JOIN team home_team ON match.home_team_id = home_team.id
		JOIN team visitor_team ON match.visitor_team_id = visitor_team.id
		WHERE match.tournament_id = $1 AND match.pool_index = $2
		ORDER BY scheduled_at
	`
	return fetchMatches(db.Query(sql, tournamentID, poolIndex))
}

func fetchMatches(rows *sql.Rows, err error) []match {
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	slice := make([]match, 0)
	for rows.Next() {
		match := match{}
		var scheduledAtStr string
		err2 := rows.Scan(&match.ID, &match.PoolIndex, &scheduledAtStr, &match.HomeTeamName, &match.VisitorTeamName, &match.HomeTeamGoals, &match.VisitorTeamGoals)
		if err2 != nil {
			panic(err2)
		}
		match.ScheduledAt = parseTime(scheduledAtStr)
		slice = append(slice, match)
	}
	return slice
}
func selectTournamentPoolRanking(db *sql.DB, tournamentID string, poolIndex int) []teamRanking {
	sql := `
		WITH finished_games AS (
			SELECT *
			FROM match 
			WHERE tournament_id =$1
			  AND pool_index=$2
			  AND home_team_goals IS NOT NULL 
			  AND visitor_team_goals IS NOT NULL
		), team_matches AS (
			SELECT team.id AS id, team.name AS name, home_team_goals AS team_goals, visitor_team_goals AS opponent_goals
			FROM team 
			JOIN finished_games ON finished_games.home_team_id = team.id 
			UNION
			SELECT team.id AS id, team.name AS name, visitor_team_goals AS team_goals, home_team_goals AS opponent_goals
			FROM team 
			JOIN finished_games ON finished_games.visitor_team_id = team.id 
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
			FROM team_matches
		), team_summary AS (
			SELECT team_result.id, COUNT(*) AS played, SUM(win) AS win_count, SUM(draw) AS draw_count , SUM(defeat) AS defeat_count, SUM(team_goals) AS goals, (SUM(team_goals) - SUM(opponent_goals)) AS goal_balance,
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
			COALESCE(played, 0),
			COALESCE(win_count, 0),
			COALESCE(draw_count, 0),
			COALESCE(defeat_count, 0),
			COALESCE(goals, 0),
			COALESCE(goal_balance, 0),
			COALESCE(points, 0),
			RANK () OVER ( ORDER BY points DESC, goal_balance DESC, name ) rank
		FROM team 
		LEFT JOIN team_summary ON team.id = team_summary.id
		WHERE team.tournament_id = $1 AND team.pool_index = $2
		ORDER BY rank	
	`
	rows, err := db.Query(sql, tournamentID, poolIndex)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	slice := make([]teamRanking, 0)
	for rows.Next() {
		row := teamRanking{}
		err2 := rows.Scan(&row.Name, &row.Played, &row.Wins, &row.Draws, &row.Defeats, &row.Goals, &row.GoalBalance, &row.Points, &row.Rank)
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
		SELECT team.id, team.name, team.pool_index
		FROM team 
		JOIN tournament ON tournament.id = team.tournament_id
		WHERE tournament.id = $1
		ORDER BY team.pool_index, team.id	
	`
	return fetchTeams(db.Query(sql, tournamentID))
}
func selectTournamentPools(db *sql.DB, tournamentID string) []pool {
	sql := `
		SELECT tournament_id, pool_index
		FROM pool
		WHERE tournament_id = $1
		ORDER BY pool_index
	`
	rows, err := db.Query(sql, tournamentID)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	slice := make([]pool, 0)
	for rows.Next() {
		row := pool{}
		err2 := rows.Scan(&row.TournamentID, &row.Index)
		if err2 != nil {
			panic(err2)
		}
		slice = append(slice, row)
	}
	return slice
}
func selectTournamentPoolTeams(db *sql.DB, tournamentID string, poolIndex int) []team {
	sql := `
		SELECT team.id, team.name, team.pool_index
		FROM team 
		JOIN tournament ON tournament.id = team.tournament_id
		WHERE tournament.id = $1 AND team.pool_index = $2
		ORDER BY team.pool_index, team.id	
	`
	return fetchTeams(db.Query(sql, tournamentID, poolIndex))
}

func fetchTeams(rows *sql.Rows, err error) []team {
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	slice := make([]team, 0)
	for rows.Next() {
		row := team{}
		err2 := rows.Scan(&row.ID, &row.Name, &row.PoolIndex)
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
func insertPool(db *sql.DB, p pool) {
	sql := `
		INSERT INTO pool(tournament_id, pool_index)
		VALUES ($1, $2)
	`
	_, err := db.Exec(sql, p.TournamentID, p.Index)
	if err != nil {
		panic(err)
	}
}
func insertTeams(db *sql.DB, tournamentID string, teams []team) {
	sql := `
		INSERT INTO team(tournament_id, name, pool_index)
		VALUES ($1, $2, $3)
	`
	for _, team := range teams {
		_, err := db.Exec(sql, tournamentID, team.Name, team.PoolIndex)
		if err != nil {
			panic(err)
		}
	}
}
func updateTeamNames(db *sql.DB, teams []team) {
	sql := `
		UPDATE team SET name = $1
		WHERE id = $2
	`
	for _, team := range teams {
		_, err := db.Exec(sql, team.Name, team.ID)
		if err != nil {
			panic(err)
		}
	}
}
func insertMatches(db *sql.DB, tournamentID string, matchs []match) {
	sql := `
		INSERT INTO match(tournament_id, pool_index, scheduled_at, home_team_id, visitor_team_id)
		VALUES ($1, $2, $3, $4, $5)
	`
	for _, match := range matchs {
		_, err := db.Exec(sql, tournamentID, match.PoolIndex, match.ScheduledAt.Format(timeFormat), match.HomeTeamID, match.VisitorTeamID)
		if err != nil {
			panic(err)
		}
	}
}
func deleteTournament(db *sql.DB, tournamentID string) {
	sql := "DELETE FROM match WHERE tournament_id = $1"
	_, err := db.Exec(sql, tournamentID)
	if err != nil {
		panic(err)
	}
	sql = "DELETE FROM team WHERE tournament_id = $1"
	_, err = db.Exec(sql, tournamentID)
	if err != nil {
		panic(err)
	}
	sql = "DELETE FROM pool WHERE tournament_id = $1"
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

func saveMatchScore(db *sql.DB, matchID int, homeTeamGoals int, visitorTeamGoals int) {
	sql := "UPDATE match SET home_team_goals=$1, visitor_team_goals=$2 WHERE id = $3"
	_, err := db.Exec(sql, homeTeamGoals, visitorTeamGoals, matchID)
	if err != nil {
		panic(err)
	}
}

func parseTime(timeStr string) time.Time {
	t, _ := time.Parse(timeFormat, timeStr)
	return t
}
