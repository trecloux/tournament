package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/echoview"
	"github.com/foolin/goview/supports/gorice"
	"github.com/thoas/go-funk"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db := initDB()

	e := echo.New()
	e.Debug = true

	e.Server.ReadTimeout = 5 * time.Second
	e.Server.WriteTimeout = 10 * time.Second
	e.Server.IdleTimeout = 120 * time.Second

	e.Renderer = &echoview.ViewEngine{
		ViewEngine: gorice.NewWithConfig(
			rice.MustFindBox("views"),
			goview.Config{
				Root:         "views",
				Extension:    ".html",
				Master:       "layouts/master",
				Partials:     []string{"partials/fragments"},
				Funcs:        make(template.FuncMap),
				DisableCache: false,
				Delims:       goview.Delims{Left: "{{", Right: "}}"},
			},
		),
	}
	e.Use(middleware.Logger())
	e.Pre(middleware.MethodOverrideWithConfig(middleware.MethodOverrideConfig{
		Getter: middleware.MethodFromForm("_method"),
	}))
	e.Use(NoCache())
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Gzip())
	e.Use(middleware.Recover())

	assetHandler := http.FileServer(rice.MustFindBox("assets").HTTPBox())
	e.GET("/assets/*", echo.WrapHandler(http.StripPrefix("/assets/", assetHandler)))

	e.GET("/", index(db))
	e.GET("/admin", admin(db))
	e.GET("/admin/tournaments/:id", adminTournament(db))
	e.POST("/admin/tournaments/:id/teams", postTeamNames(db))
	e.GET("/admin/tournaments/:id/pools-matches", poolsMatchesScores(db))
	e.GET("/admin/tournaments/:id/ranking-matches", rankingMatchesScores(db))
	e.GET("/tournaments/:id/matches", getAllTournamentMatches(db))
	e.GET("/tournaments/:id/pools/matches", getAllTournamentPoolsMatches(db))
	e.GET("/tournaments/:id/pools/:poolIndex/matches", getPoolMatches(db))
	e.GET("/tournaments/:id/pools/ranking", getAllTournamentPoolsRanking(db))
	e.GET("/tournaments/:id/pools/:poolIndex/ranking", getPoolRanking(db))
	e.GET("/tournaments/:id/ranking-matches", getTournamentRankingMatches(db))
	e.GET("/tournaments/:id/final-ranking", getFinalRanking(db))
	e.POST("/tournaments", createTournament(db))
	e.DELETE("/tournaments/:id", removeTournament(db))
	e.POST("/tournaments/:tournamentId/pools/:poolIndex/matches/:matchId/score", postPoolMatchScore(db))
	e.POST("/tournaments/:tournamentId/ranking-matches/:key/score", postRankingMatchScore(db))

	address := ":8080"
	if value, ok := os.LookupEnv("PORT"); ok {
		address = ":" + value
	}
	e.Logger.Fatal(e.Start(address))
}

func index(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournaments := selectTournaments(db)
		tournaments = funk.Map(tournaments, func(tournament tournament) tournament {
			tournament.Pools = selectTournamentPools(db, tournament.ID)
			return tournament
		}).([]tournament)
		return c.Render(http.StatusOK, "index", echo.Map{"title": "Tounois", "tournaments": tournaments})
	}
}
func admin(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournaments := selectTournaments(db)
		tournaments = funk.Map(tournaments, func(tournament tournament) tournament {
			tournament.Pools = selectTournamentPools(db, tournament.ID)
			return tournament
		}).([]tournament)
		return c.Render(http.StatusOK, "admin/index", echo.Map{"title": "Admin", "tournaments": tournaments})
	}
}
func adminTournament(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		teams := selectTournamentTeams(db, tournamentID)
		matches := selectAllTournamentPoolMatches(db, tournamentID)
		return c.Render(
			http.StatusOK,
			"admin/tournament",
			echo.Map{"title": "Scores", "tournament": tournament, "teams": teams, "matches": matches},
		)
	}
}
func poolsMatchesScores(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		pools := loadAllPoolsMatches(db, tournamentID)
		return c.Render(http.StatusOK, "admin/pools-matches", echo.Map{
			"title":      "Scores",
			"tournament": tournament,
			"pools":      pools,
		})
	}
}
func rankingMatchesScores(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		rankingMatches := tournamentRankingMatches(db, tournamentID)
		return c.Render(http.StatusOK, "admin/ranking-matches", echo.Map{
			"title":          "Scores",
			"tournament":     tournament,
			"rankingMatches": rankingMatches,
			"invalidScore":	  c.FormValue("error") == "invalid_score",
		})
	}
}

func getAllTournamentMatches(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		pools := loadAllPoolsMatches(db, tournamentID)
		rankingMatches := tournamentRankingMatches(db, tournamentID)
		return c.Render(http.StatusOK, "all-matches", echo.Map{
			"title":          "Rencontres",
			"tournament":     tournament,
			"pools":          pools,
			"rankingMatches": rankingMatches,
		})
	}
}
func getAllTournamentPoolsMatches(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		pools := loadAllPoolsMatches(db, tournamentID)
		return c.Render(http.StatusOK, "pools-matches", echo.Map{
			"title":      "Rencontres",
			"tournament": tournament,
			"pools":      pools,
		})
	}
}
func getTournamentRankingMatches(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		matches := tournamentRankingMatches(db, tournamentID)
		return c.Render(http.StatusOK, "ranking-matches", echo.Map{
			"title":          "Rencontres",
			"tournament":     tournament,
			"rankingMatches": matches,
		})
	}
}

func getPoolMatches(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		poolIndex, _ := strconv.Atoi(c.Param("poolIndex"))
		_pool := selectTournamentPool(db, tournamentID, poolIndex)
		poolMatches := loadPoolMatches(db, _pool)
		return c.Render(http.StatusOK, "pool-matches", echo.Map{
			"title":      "Rencontres poule" + _pool.Name,
			"tournament": tournament,
			"pool":       poolMatches,
		})
	}
}

func tournamentRankingMatches(db *sql.DB, tournamentID string) []rankingMatch {
	matches := selectTournamentRankingMatches(db, tournamentID)
	pools := selectTournamentPools(db, tournamentID)
	matches = funk.Map(matches, func(match rankingMatch) rankingMatch {
		match.ValidTeams = match.HomeTeamName.Valid && match.VisitorTeamName.Valid
		if match.HomeTeamGoals.Valid && match.VisitorTeamGoals.Valid && match.HomeTeamGoals.Int64 == match.VisitorTeamGoals.Int64 {
			if match.WinnerTeamID.Int64 == match.HomeTeamID.Int64 {
				match.PenaltyShootOutWinner = "home"
			} else {
				match.PenaltyShootOutWinner = "visitor"
			}
		} else {
			match.PenaltyShootOutWinner = "none"
		}
		if !match.HomeTeamName.Valid {
			match.HomeTeamName = rankingMatchTeamName(pools, match.HomeTeamPoolIndex, match.HomeTeamPoolRank, match.HomeTeamSourceRankingMatch, match.HomeTeamSourceRankingMatchWinner)
		}
		if !match.VisitorTeamName.Valid {
			match.VisitorTeamName = rankingMatchTeamName(pools, match.VisitorTeamPoolIndex, match.VisitorTeamPoolRank, match.VisitorTeamSourceRankingMatch, match.VisitorTeamSourceRankingMatchWinner)
		}
		return match
	}).([]rankingMatch)
	return matches
}
func rankingMatchTeamName(pools []pool, poolIndex sql.NullInt64, poolRank sql.NullInt64, rankingMatchKey sql.NullString, rankingMatchWinner sql.NullBool) sql.NullString {
	var name string
	if poolIndex.Valid {
		pool := funk.Find(pools, func(p pool) bool {
			return p.Index == int(poolIndex.Int64)
		}).(pool)
		var prefix string
		if poolRank.Int64 == 1 {
			prefix = "1er"
		} else {
			prefix = fmt.Sprintf("%deme", poolRank.Int64)
		}
		name = fmt.Sprintf("%s poule %s", prefix, pool.Name)
	} else {
		var prefix string
		if rankingMatchWinner.Bool {
			prefix = "Gagnant"
		} else {
			prefix = "Perdant"
		}
		name = fmt.Sprintf("%s match %s", prefix, rankingMatchKey.String)
	}
	return sql.NullString{String: name, Valid: true}
}
func loadAllPoolsMatches(db *sql.DB, tournamentID string) []poolViewModel {
	pools := selectTournamentPools(db, tournamentID)
	poolViews := make([]poolViewModel, 0)
	for _, pool := range pools {
		poolViews = append(poolViews, loadPoolMatches(db, pool))
	}
	return poolViews
}

func loadPoolMatches(db *sql.DB, pool pool) poolViewModel {
	return poolViewModel{
		PoolIndex: pool.Index,
		PoolName:  pool.Name,
		Matches:   selectTournamentPoolMatches(db, pool.TournamentID, pool.Index),
	}
}
func getAllTournamentPoolsRanking(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		//finalRanking := selectTournamentFinalRanking(db, tournamentID)
		return c.Render(http.StatusOK, "pools-ranking", echo.Map{
			"title":      "Classements",
			"tournament": tournament,
			"pools":      loadAllTournamentPoolsRanking(db, tournamentID),
		})
	}
}
func getFinalRanking(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		finalRanking := selectTournamentFinalRanking(db, tournamentID)
		return c.Render(http.StatusOK, "final-ranking", echo.Map{
			"title":      "Classements",
			"tournament": tournament,
			"ranking":    finalRanking,
		})
	}
}
func getPoolRanking(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		poolIndex, _ := strconv.Atoi(c.Param("poolIndex"))
		pool := selectTournamentPool(db, tournamentID, poolIndex)
		return c.Render(http.StatusOK, "pool-ranking", echo.Map{
			"title":      "Classement poule" + pool.Name,
			"tournament": tournament,
			"ranking":    loadPoolRanking(db, tournamentID, pool),
		})
	}
}
func loadAllTournamentPoolsRanking(db *sql.DB, tournamentID string) []rankingViewModel {
	pools := selectTournamentPools(db, tournamentID)
	rankingViews := make([]rankingViewModel, 0)
	for _, pool := range pools {
		rankingViews = append(rankingViews, loadPoolRanking(db, tournamentID, pool))
	}
	return rankingViews
}

func loadPoolRanking(db *sql.DB, tournamentID string, pool pool) rankingViewModel {
	return rankingViewModel{
		PoolIndex:    pool.Index,
		PoolName:     pool.Name,
		TeamRankings: selectTournamentPoolRanking(db, tournamentID, pool.Index),
	}
}
func removeTournament(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		deleteTournament(db, tournamentID)
		return c.Redirect(http.StatusSeeOther, "/admin")
	}
}

func createTournament(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		startTime, _ := time.Parse("15:04", c.FormValue("startTime"))
		gameDuration, _ := time.ParseDuration(c.FormValue("gameDurationMinutes") + "m")
		betweenGamesDuration, _ := time.ParseDuration(c.FormValue("betweenGamesDurationMinutes") + "m")
		tournamentID := c.FormValue("id")
		tournamentName := c.FormValue("name")
		nbTeams, _ := strconv.Atoi(c.FormValue("nbTeams"))
		nbPools, _ := strconv.Atoi(c.FormValue("nbPools"))
		pointsPerWin, _ := strconv.ParseFloat(c.FormValue("pointsPerWin"), 64)
		pointsPerDraw, _ := strconv.ParseFloat(c.FormValue("pointsPerDraw"), 64)
		pointsPerDefeat, _ := strconv.ParseFloat(c.FormValue("pointsPerDefeat"), 64)
		pointsPerGoal, _ := strconv.ParseFloat(c.FormValue("pointsPerGoal"), 64)
		tournament := tournament{
			ID:              tournamentID,
			Name:            tournamentName,
			pointsPerWin:    pointsPerWin,
			pointsPerDraw:   pointsPerDraw,
			pointsPerDefeat: pointsPerDefeat,
			pointsPerGoal:   pointsPerGoal,
		}
		insertTournament(db, tournament)

		nbTeamsPerPool := nbTeams / nbPools
		nbTeamsToDispatch := nbTeams % nbPools

		teamIndex := 1
		for poolIndex := 1; poolIndex <= nbPools; poolIndex++ {
			poolSize := nbTeamsPerPool
			if nbTeamsToDispatch > 0 {
				poolSize++
				nbTeamsToDispatch--
			}
			currentPool := pool{
				TournamentID: tournamentID,
				Index:        poolIndex,
			}
			poolTeams := make([]team, 0)
			for j := 1; j <= poolSize; j++ {
				team := team{
					Name:      fmt.Sprintf("Team %d", teamIndex),
					PoolIndex: poolIndex,
				}
				poolTeams = append(poolTeams, team)
				teamIndex++
			}
			insertPool(db, currentPool)
			insertTeams(db, tournamentID, poolTeams)
			poolTeams = selectTournamentPoolTeams(db, tournamentID, poolIndex)
			pairs := roundRobin(poolTeams)
			matches := make([]poolMatch, 0)
			matchTime := startTime
			for _, pair := range pairs {
				match := poolMatch{
					PoolIndex:     poolIndex,
					ScheduledAt:   matchTime,
					HomeTeamID:    pair.Home.ID,
					VisitorTeamID: pair.Visitor.ID,
				}
				matches = append(matches, match)
				matchTime = matchTime.Add(gameDuration)
				matchTime = matchTime.Add(betweenGamesDuration)
			}
			insertPoolMatches(db, tournamentID, matches)
		}
		return c.Redirect(http.StatusSeeOther, "/admin/tournaments/"+tournamentID)
	}
}
func postTeamNames(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		existingTeams := selectTournamentTeams(db, tournamentID)
		updatedTeams := make([]team, 0)
		for _, team := range existingTeams {
			team.Name = c.FormValue(fmt.Sprintf("team_%d", team.ID))
			updatedTeams = append(updatedTeams, team)
		}
		updateTeamNames(db, updatedTeams)
		return c.Redirect(http.StatusSeeOther, "/admin/tournaments/"+tournamentID)
	}

}
func postPoolMatchScore(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("tournamentId")
		matchID, _ := strconv.Atoi(c.Param("matchId"))
		poolIndex, _ := strconv.Atoi(c.Param("poolIndex"))
		homeTeamGoals, _ := strconv.Atoi(c.FormValue("homeTeamGoals"))
		visitorTeamGoals, _ := strconv.Atoi(c.FormValue("visitorTeamGoals"))
		savePoolMatchScore(db, tournamentID, matchID, homeTeamGoals, visitorTeamGoals)
		matchesToBePlayed := countPoolMatchesToBePlayed(db, tournamentID, poolIndex)
		if matchesToBePlayed == 0 {
			teamRanking := selectTournamentPoolRanking(db, tournamentID, poolIndex)
			for _, teamRank := range teamRanking {
				updateRankingMatchFromPoolRank(db, tournamentID, poolIndex, teamRank.Rank, teamRank.ID)
			}
		}
		return c.Redirect(http.StatusSeeOther, "/admin/tournaments/"+tournamentID+"/pools-matches#"+c.FormValue("anchor"))
	}
}
func postRankingMatchScore(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("tournamentId")
		key := c.Param("key")
		homeTeamGoals, _ := strconv.Atoi(c.FormValue("homeTeamGoals"))
		visitorTeamGoals, _ := strconv.Atoi(c.FormValue("visitorTeamGoals"))
		penaltyShootOutWinner := c.FormValue("penaltyShootOutWinner")
		if !validRankingMatchScore(homeTeamGoals, visitorTeamGoals, penaltyShootOutWinner) {
			return c.Redirect(http.StatusSeeOther, "/admin/tournaments/"+tournamentID+"/ranking-matches?error=invalid_score")
		}
		homeTeamID, visitorTeamID := selectRankingMatchTeamIDs(db, tournamentID, key)
		var winnerTeamID int
		var looserTeamID int
		if homeTeamGoals > visitorTeamGoals || penaltyShootOutWinner == "home" {
			winnerTeamID = homeTeamID
			looserTeamID = visitorTeamID
		} else if homeTeamGoals < visitorTeamGoals  || penaltyShootOutWinner == "visitor" {
			winnerTeamID = visitorTeamID
			looserTeamID = homeTeamID
		}
		saveRankingMatchScore(db, tournamentID, key, homeTeamGoals, visitorTeamGoals, winnerTeamID, looserTeamID)
		updateRankingMatchFromSourceRankingMatch(db, tournamentID, key, winnerTeamID, looserTeamID)
		return c.Redirect(http.StatusSeeOther, "/admin/tournaments/"+tournamentID+"/ranking-matches")
	}
}

func validRankingMatchScore(homeTeamGoals int, visitorTeamGoals int, penaltyShootOutWinner string) bool {
	if homeTeamGoals == visitorTeamGoals {
		return penaltyShootOutWinner == "home" || penaltyShootOutWinner == "visitor"
	} else {
		return penaltyShootOutWinner == "none"
	}
}
