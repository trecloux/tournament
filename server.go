package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	rice "github.com/GeertJohan/go.rice"
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

	viewEngine := gorice.New(rice.MustFindBox("views"))
	e.Renderer = &echoview.ViewEngine{
		ViewEngine: viewEngine,
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
	e.GET("/admin/tournaments/:id/matchs", adminMatches(db))
	e.GET("/tournaments/:id/matchs", getMatches(db))
	e.GET("/tournaments/:id/ranking", getRanking(db))
	e.POST("/tournaments", createTournament(db))
	e.DELETE("/tournaments/:id", removeTournament(db))
	e.POST("/tournaments/:tournamentId/pools/:poolIndex/matchs/:matchId/score", postPoolMatchScore(db))
	e.POST("/tournaments/:tournamentId/ranking-matchs/:key/score", postRankingMatchScore(db))

	address := ":8080"
	if value, ok := os.LookupEnv("PORT"); ok {
		address = ":" + value
	}
	e.Logger.Fatal(e.Start(address))
}

func index(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournaments := selectTournaments(db)
		return c.Render(http.StatusOK, "index", echo.Map{"title": "Tounois", "tournaments": tournaments})
	}
}
func admin(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournaments := selectTournaments(db)
		return c.Render(http.StatusOK, "admin/index", echo.Map{"title": "Admin", "tournaments": tournaments})
	}
}
func adminTournament(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		teams := selectTournamentTeams(db, tournamentID)
		matchs := selectAllTournamentPoolMatchs(db, tournamentID)
		return c.Render(
			http.StatusOK,
			"admin/tournament",
			echo.Map{"title": "Scores", "tournament": tournament, "teams": teams, "matchs": matchs},
		)
	}
}
func adminMatches(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		pools := poolsMatchs(db, tournamentID)
		rankingMatchs := tournamentRankingMatchs(db, tournamentID)
		return c.Render(http.StatusOK, "admin/matchs", echo.Map{
			"title":         "Scores",
			"tournament":    tournament,
			"pools":         pools,
			"rankingMatchs": rankingMatchs,
		})
	}
}
func getMatches(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		pools := poolsMatchs(db, tournamentID)
		rankingMatchs := tournamentRankingMatchs(db, tournamentID)
		return c.Render(http.StatusOK, "matchs", echo.Map{
			"title":         "Rencontres",
			"tournament":    tournament,
			"pools":         pools,
			"rankingMatchs": rankingMatchs,
		})
	}
}
func tournamentRankingMatchs(db *sql.DB, tournamentID string) []rankingMatch {
	matchs := selectTournamentRankingMatchs(db, tournamentID)
	pools := selectTournamentPools(db, tournamentID)
	matchs = funk.Map(matchs, func(match rankingMatch) rankingMatch {
		if !match.HomeTeamName.Valid {
			match.HomeTeamName = rankingMatchTeamName(pools, match.HomeTeamPoolIndex, match.HomeTeamPoolRank, match.HomeTeamSourceRankingMatch, match.HomeTeamSourceRankingMatchWinner)
		}
		if !match.VisitorTeamName.Valid {
			match.VisitorTeamName = rankingMatchTeamName(pools, match.VisitorTeamPoolIndex, match.VisitorTeamPoolRank, match.VisitorTeamSourceRankingMatch, match.VisitorTeamSourceRankingMatchWinner)
		}
		fmt.Printf("Match            : %s\n", match.Key)
		fmt.Printf("Home    team name: %s\n", match.HomeTeamName.String)
		fmt.Printf("Visitor team name: %s\n", match.VisitorTeamName.String)
		return match
	}).([]rankingMatch)
	return matchs
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
func poolsMatchs(db *sql.DB, tournamentID string) []poolViewModel {
	pools := selectTournamentPools(db, tournamentID)
	poolViews := make([]poolViewModel, 0)
	for _, pool := range pools {
		poolViews = append(poolViews, poolViewModel{
			PoolIndex: pool.Index,
			PoolName:  pool.Name,
			Matchs:    selectTournamentPoolMatchs(db, tournamentID, pool.Index),
		})
	}
	return poolViews
}
func getRanking(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		finalRanking := selectTournamentFinalRanking(db, tournamentID)
		return c.Render(http.StatusOK, "ranking", echo.Map{
			"title":        "Classements",
			"tournament":   tournament,
			"pools":        poolsRankings(db, tournamentID),
			"finalRanking": finalRanking,
		})
	}
}
func poolsRankings(db *sql.DB, tournamentID string) []rankingViewModel {
	pools := selectTournamentPools(db, tournamentID)
	rankingViews := make([]rankingViewModel, 0)
	for _, pool := range pools {
		rankingViews = append(rankingViews, rankingViewModel{
			PoolIndex:    pool.Index,
			PoolName:     pool.Name,
			TeamRankings: selectTournamentPoolRanking(db, tournamentID, pool.Index),
		})
	}
	return rankingViews
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
			matchs := make([]poolMatch, 0)
			matchTime := startTime
			for _, pair := range pairs {
				match := poolMatch{
					PoolIndex:     poolIndex,
					ScheduledAt:   matchTime,
					HomeTeamID:    pair.Home.ID,
					VisitorTeamID: pair.Visitor.ID,
				}
				matchs = append(matchs, match)
				matchTime = matchTime.Add(gameDuration)
				matchTime = matchTime.Add(betweenGamesDuration)
			}
			insertMatchs(db, tournamentID, matchs)
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
		savePoolMatchScore(db, matchID, homeTeamGoals, visitorTeamGoals)
		matchsToBePlayed := countPoolMatchsToBePlayed(db, tournamentID, poolIndex)
		if matchsToBePlayed == 0 {
			teamRanking := selectTournamentPoolRanking(db, tournamentID, poolIndex)
			for _, teamRank := range teamRanking {
				updateRankingMatchFromPoolRank(db, tournamentID, poolIndex, teamRank.Rank, teamRank.ID)
			}
		}
		return c.Redirect(http.StatusSeeOther, "/admin/tournaments/"+tournamentID+"/matchs")
	}
}
func postRankingMatchScore(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("tournamentId")
		key := c.Param("key")
		homeTeamGoals, _ := strconv.Atoi(c.FormValue("homeTeamGoals"))
		visitorTeamGoals, _ := strconv.Atoi(c.FormValue("visitorTeamGoals"))
		saveRankingMatchScore(db, tournamentID, key, homeTeamGoals, visitorTeamGoals)
		homeTeamID, visitorTeamID := selectRankingMatchTeamIDs(db, tournamentID, key)
		var winnerTeamID int
		var looserTeamID int
		if homeTeamGoals > visitorTeamGoals {
			winnerTeamID = homeTeamID
			looserTeamID = visitorTeamID
		} else {
			winnerTeamID = visitorTeamID
			looserTeamID = homeTeamID
		}
		updateRankingMatchFromSourceRankingMatch(db, tournamentID, key, winnerTeamID, looserTeamID)
		return c.Redirect(http.StatusSeeOther, "/admin/tournaments/"+tournamentID+"/matchs")
	}
}
