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
					
					CREATE TABLE pitch (
						id INTEGER,
						name TEXT NOT NULL,
						tournament_id TEXT NOT NULL REFERENCES tournament(id),
						PRIMARY KEY(id, tournament_id)
					);
					
					CREATE TABLE pool (
						tournament_id TEXT NOT NULL REFERENCES tournament(id),
						pool_index INTEGER NOT NULL,
						name TEXT NOT NULL,
						PRIMARY KEY(tournament_id, pool_index)
					);
					
					CREATE TABLE team (
						id INTEGER,
						tournament_id TEXT NOT NULL REFERENCES tournament(id),
						pool_index INTEGER NOT NULL REFERENCES pool(pool_index),
						name TEXT NOT NULL,
						PRIMARY KEY(id, tournament_id)
					);
					
					CREATE TABLE pool_match (
						id INTEGER,
						tournament_id TEXT NOT NULL REFERENCES tournament(id),
						pool_index INTEGER NOT NULL REFERENCES pool(pool_index),
						scheduled_at INTEGER NOT NULL,
						pitch_id INTEGER NOT NULL REFERENCES pitch(id),
						home_team_id INTEGER NOT NULL REFERENCES team(id),
						visitor_team_id INTEGER NOT NULL REFERENCES team(id),
						home_team_goals INTEGER,
						visitor_team_goals INTEGER,
						PRIMARY KEY(id, tournament_id, pool_index)						
					);
					
					CREATE TABLE ranking_match (
						key TEXT NOT NULL,
						tournament_id TEXT NOT NULL REFERENCES tournament(id),
						scheduled_at INTEGER NOT NULL,
						pitch_id INTEGER NOT NULL REFERENCES pitch(id),
						home_team_pool_index INTEGER REFERENCES pool(pool_index),
						home_team_pool_rank INTEGER,
						home_team_source_ranking_match INTEGER,
						home_team_source_ranking_match_winner BOOLEAN,
						home_team_id INTEGER REFERENCES team(id),
						home_team_goals INTEGER,
						visitor_team_pool_index INTEGER REFERENCES pool(pool_index),
						visitor_team_pool_rank INTEGER,
						visitor_team_source_ranking_match INTEGER,
						visitor_team_source_ranking_match_winner BOOLEAN,
						visitor_team_id INTEGER REFERENCES team(id),
						visitor_team_goals INTEGER,
						winner_team_id INTEGER REFERENCES team(id),
						looser_team_id INTEGER REFERENCES team(id),
						looser_final_rank INTEGER,
						winner_final_rank INTEGER,
						PRIMARY KEY(key, tournament_id)
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

func selectAllTournamentPoolMatches(db *sql.DB, tournamentID string) []poolMatch {
	sql := `
		SELECT match.id, match.pool_index, match.scheduled_at, home_team.name, visitor_team.name, match.home_team_goals, match.visitor_team_goals, pitch.name AS pitch_name
		FROM pool_match match 
		JOIN team home_team ON match.home_team_id = home_team.id AND home_team.tournament_id = $1
		JOIN team visitor_team ON match.visitor_team_id = visitor_team.id AND visitor_team.tournament_id = $1
		JOIN pitch ON match.pitch_id = pitch.id AND pitch.tournament_id = $1
		WHERE match.tournament_id = $1
		ORDER BY scheduled_at
	`
	return fetchPoolMatches(db.Query(sql, tournamentID))
}

func selectTournamentPoolMatches(db *sql.DB, tournamentID string, poolIndex int) []poolMatch {
	sql := `
		SELECT match.id, match.pool_index, match.scheduled_at, home_team.name, visitor_team.name, match.home_team_goals, match.visitor_team_goals, pitch.name AS pitch_name
		FROM pool_match match 
		JOIN team home_team ON match.home_team_id = home_team.id AND home_team.tournament_id = $1
		JOIN team visitor_team ON match.visitor_team_id = visitor_team.id AND visitor_team.tournament_id = $1
		JOIN pitch ON match.pitch_id = pitch.id AND pitch.tournament_id = $1
		WHERE match.tournament_id = $1 AND match.pool_index = $2
		ORDER BY scheduled_at
	`
	return fetchPoolMatches(db.Query(sql, tournamentID, poolIndex))
}

func fetchPoolMatches(rows *sql.Rows, err error) []poolMatch {
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	slice := make([]poolMatch, 0)
	for rows.Next() {
		match := poolMatch{}
		var scheduledAtStr string
		err2 := rows.Scan(&match.ID, &match.PoolIndex, &scheduledAtStr, &match.HomeTeamName, &match.VisitorTeamName, &match.HomeTeamGoals, &match.VisitorTeamGoals, &match.PitchName)
		if err2 != nil {
			panic(err2)
		}
		match.ScheduledAt = parseTime(scheduledAtStr)
		slice = append(slice, match)
	}
	return slice
}

func selectTournamentRankingMatches(db *sql.DB, tournamentID string) []rankingMatch {
	sql := `
		SELECT match.key, scheduled_at,
			home_team.name,    home_team_pool_index,    home_team_pool_rank,    home_team_source_ranking_match,    home_team_source_ranking_match_winner,    home_team_goals,    home_team_id,
			visitor_team.name, visitor_team_pool_index, visitor_team_pool_rank, visitor_team_source_ranking_match, visitor_team_source_ranking_match_winner, visitor_team_goals, visitor_team_id,
			winner_team_id, looser_team_id,
			pitch.name AS pitch_name
		FROM ranking_match match 
		JOIN pitch ON match.pitch_id = pitch.id AND pitch.tournament_id = $1
		LEFT JOIN team home_team ON match.home_team_id = home_team.id AND home_team.tournament_id = $1
		LEFT JOIN team visitor_team ON match.visitor_team_id = visitor_team.id AND visitor_team.tournament_id = $1
		WHERE match.tournament_id = $1
		ORDER BY scheduled_at
	`
	rows, err := db.Query(sql, tournamentID)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	slice := make([]rankingMatch, 0)
	for rows.Next() {
		match := rankingMatch{}
		var scheduledAtStr string
		err2 := rows.Scan(&match.Key, &scheduledAtStr,
			&match.HomeTeamName,
			&match.HomeTeamPoolIndex, &match.HomeTeamPoolRank, &match.HomeTeamSourceRankingMatch, &match.HomeTeamSourceRankingMatchWinner,
			&match.HomeTeamGoals, &match.HomeTeamID,
			&match.VisitorTeamName,
			&match.VisitorTeamPoolIndex, &match.VisitorTeamPoolRank, &match.VisitorTeamSourceRankingMatch, &match.VisitorTeamSourceRankingMatchWinner,
			&match.VisitorTeamGoals, &match.VisitorTeamID,
			&match.WinnerTeamID, &match.LooserTeamID,
			&match.PitchName)
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
			FROM pool_match 
			WHERE tournament_id =$1
			  AND pool_index=$2
			  AND home_team_goals IS NOT NULL 
			  AND visitor_team_goals IS NOT NULL
		), team_matches AS (
			SELECT team.id AS id, team.name AS name, finished_games.id, home_team_goals AS team_goals, visitor_team_goals AS opponent_goals
			FROM team 
			JOIN finished_games ON finished_games.home_team_id = team.id AND finished_games.tournament_id = team.tournament_id
			UNION
			SELECT team.id AS id, team.name AS name, finished_games.id, visitor_team_goals AS team_goals, home_team_goals AS opponent_goals
			FROM team 
			JOIN finished_games ON finished_games.visitor_team_id = team.id AND finished_games.tournament_id = team.tournament_id
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
		  team.id,
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
		err2 := rows.Scan(&row.ID, &row.Name, &row.Played, &row.Wins, &row.Draws, &row.Defeats, &row.Goals, &row.GoalBalance, &row.Points, &row.Rank)
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
		SELECT tournament_id, pool_index, name
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
		err2 := rows.Scan(&row.TournamentID, &row.Index, &row.Name)
		if err2 != nil {
			panic(err2)
		}
		slice = append(slice, row)
	}
	return slice
}
func selectTournamentPool(db *sql.DB, tournamentID string, poolIndex int) pool {
	sql := `
		SELECT tournament_id, pool_index, name
		FROM pool
		WHERE tournament_id = $1 AND pool_index=$2
	`
	rows, err := db.Query(sql, tournamentID, poolIndex)
	if err != nil {
		panic(err)
	}
	rows.Next()
	row := pool{}
	err2 := rows.Scan(&row.TournamentID, &row.Index, &row.Name)
	if err2 != nil {
		panic(err2)
	}
	return row
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
func insertPoolMatches(db *sql.DB, tournamentID string, matches []poolMatch) {
	sql := `
		INSERT INTO match(tournament_id, pool_index, scheduled_at, home_team_id, visitor_team_id)
		VALUES ($1, $2, $3, $4, $5)
	`
	for _, match := range matches {
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

func savePoolMatchScore(db *sql.DB, tournamentID string,  matchID int, homeTeamGoals int, visitorTeamGoals int) {
	sql := "UPDATE pool_match SET home_team_goals=$1, visitor_team_goals=$2 WHERE tournament_id = $3 AND id = $4"
	_, err := db.Exec(sql, homeTeamGoals, visitorTeamGoals, tournamentID,  matchID)
	if err != nil {
		panic(err)
	}
}

func saveRankingMatchScore(db *sql.DB, tournamentID string, key string, homeTeamGoals int, visitorTeamGoals int, winnerTeamID int, looserTeamID int) {
	sql := "UPDATE ranking_match SET home_team_goals=$1, visitor_team_goals=$2, winner_team_id=$3, looser_team_id=$4 WHERE tournament_id=$5 AND key = $6"
	_, err := db.Exec(sql, homeTeamGoals, visitorTeamGoals, winnerTeamID, looserTeamID, tournamentID, key)
	if err != nil {
		panic(err)
	}
}

func countPoolMatchesToBePlayed(db *sql.DB, tournamentID string, poolIndex int) int {
	sql := `
	SELECT COUNT(*)
	FROM pool_match
	WHERE tournament_id = $1
		AND pool_index = $2
		AND (home_team_goals IS NULL OR visitor_team_goals IS NULL)
	`
	rows, err := db.Query(sql, tournamentID, poolIndex)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	rows.Next()
	var count int
	err = rows.Scan(&count)
	if err != nil {
		panic(err)
	}
	return count
}

func selectRankingMatchTeamIDs(db *sql.DB, tournamentID string, key string) (int, int) {
	sql := `
	SELECT home_team_id, visitor_team_id
	FROM ranking_match
	WHERE tournament_id = $1
		AND key = $2
	`
	rows, err := db.Query(sql, tournamentID, key)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	rows.Next()
	var homeTeamID int
	var visitorTeamID int
	err = rows.Scan(&homeTeamID, &visitorTeamID)
	if err != nil {
		panic(err)
	}
	return homeTeamID, visitorTeamID
}

func updateRankingMatchFromPoolRank(db *sql.DB, tournamentID string, poolIndex int, poolRank int, teamID int) {
	sql := `
	UPDATE ranking_match 
	SET home_team_id=$1
	WHERE tournament_id = $2
	 AND home_team_pool_index = $3
	 AND home_team_pool_rank = $4
	`
	_, err := db.Exec(sql, teamID, tournamentID, poolIndex, poolRank)
	if err != nil {
		panic(err)
	}
	sql = `
	UPDATE ranking_match 
	SET visitor_team_id=$1
	WHERE tournament_id = $2
	 AND visitor_team_pool_index = $3
	 AND visitor_team_pool_rank = $4
	`
	_, err = db.Exec(sql, teamID, tournamentID, poolIndex, poolRank)
	if err != nil {
		panic(err)
	}

}

func updateRankingMatchFromSourceRankingMatch(db *sql.DB, tournamentID string, sourceMatchKey string, winnerTeamID int, looserTeamID int) {
	_, err := db.Exec("UPDATE ranking_match SET home_team_id=$1 WHERE tournament_id = $2 AND home_team_source_ranking_match = $3 AND home_team_source_ranking_match_winner = true",
		winnerTeamID, tournamentID, sourceMatchKey)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("UPDATE ranking_match SET home_team_id=$1 WHERE tournament_id = $2 AND home_team_source_ranking_match = $3 AND home_team_source_ranking_match_winner = false",
		looserTeamID, tournamentID, sourceMatchKey)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("UPDATE ranking_match SET visitor_team_id=$1 WHERE tournament_id = $2 AND visitor_team_source_ranking_match = $3 AND visitor_team_source_ranking_match_winner = true",
		winnerTeamID, tournamentID, sourceMatchKey)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("UPDATE ranking_match SET visitor_team_id=$1 WHERE tournament_id = $2  AND visitor_team_source_ranking_match = $3 AND visitor_team_source_ranking_match_winner = false",
		looserTeamID, tournamentID, sourceMatchKey)
	if err != nil {
		panic(err)
	}
}

func selectTournamentFinalRanking(db *sql.DB, tournamentID string) []tournamentFinalRanking {
	sql := `
	WITH final_match AS (
		SELECT * FROM ranking_match WHERE tournament_id = $1 AND winner_final_rank IS NOT NULL
	), ranked_team AS (
		SELECT winner_final_rank AS rank, winner_team_id AS team_id
		FROM final_match
		WHERE winner_team_id IS NOT NULL
		UNION
		SELECT looser_final_rank AS rank, looser_team_id AS team_id
		FROM final_match
		WHERE looser_team_id IS NOT NULL
	), ranks AS (
		SELECT winner_final_rank AS rank FROM ranking_match WHERE tournament_id = $1 AND winner_final_rank IS NOT NULL
		UNION
		SELECT looser_final_rank AS rank FROM ranking_match WHERE tournament_id = $1 AND looser_final_rank IS NOT NULL
	)
	SELECT ranks.rank, team.name
	FROM ranks
	LEFT JOIN ranked_team ON ranks.rank = ranked_team.rank
	LEFT JOIN team ON team.tournament_id = $1 AND team.id=ranked_team.team_id
	ORDER by ranks.rank 
	`
	rows, err := db.Query(sql, tournamentID)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	slice := make([]tournamentFinalRanking, 0)
	for rows.Next() {
		row := tournamentFinalRanking{}
		err2 := rows.Scan(&row.Rank, &row.TeamName)
		if err2 != nil {
			panic(err2)
		}
		slice = append(slice, row)
	}
	return slice
}

func parseTime(timeStr string) time.Time {
	t, _ := time.Parse(timeFormat, timeStr)
	return t
}
