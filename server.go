package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/foolin/goview/supports/echoview"
	"github.com/foolin/goview/supports/gorice"

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
	e.GET("/admin/tournaments/:id/matches", adminMatches(db))
	e.GET("/tournaments/:id/matches", getMatches(db))
	e.GET("/tournaments/:id/ranking", getRanking(db))
	e.POST("/tournaments", createTournament(db))
	e.DELETE("/tournaments/:id", removeTournament(db))
	e.POST("/tournaments/:tournamentId/matches/:matchId/score", postScore(db))

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
		matches := selectTournamentMatches(db, tournamentID)
		return c.Render(
			http.StatusOK,
			"admin/tournament",
			echo.Map{"title": "Scores", "tournament": tournament, "teams": teams, "matches": matches},
		)
	}
}
func adminMatches(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		pools := poolsAndMatches(db, tournamentID)
		return c.Render(http.StatusOK, "admin/matches", echo.Map{"title": "Scores", "tournament": tournament, "pools": pools})
	}
}
func getMatches(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		pools := poolsAndMatches(db, tournamentID)
		return c.Render(http.StatusOK, "matches", echo.Map{"title": "Rencontres", "tournament": tournament, "pools": pools})
	}
}

func poolsAndMatches(db *sql.DB, tournamentID string) []echo.Map {
	pools := selectTournamentPools(db, tournamentID)
	poolViews := make([]echo.Map, 0)
	for _, pool := range pools {
		poolViews = append(poolViews, echo.Map{
			"poolIndex": pool.Index,
			"matches":   selectTournamentPoolMatches(db, tournamentID, pool.Index),
		})
	}
	return poolViews
}
func getRanking(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		pools := selectTournamentPools(db, tournamentID)
		rankingViews := make([]echo.Map, 0)
		for _, pool := range pools {
			rankingViews = append(rankingViews, echo.Map{
				"poolIndex": pool.Index,
				"ranking":   selectTournamentPoolRanking(db, tournamentID, pool.Index),
			})
		}
		return c.Render(http.StatusOK, "ranking", echo.Map{"title": "Classements", "tournament": tournament, "pools": rankingViews})
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
			matches := make([]match, 0)
			matchTime := startTime
			for _, pair := range pairs {
				match := match{
					PoolIndex:     poolIndex,
					ScheduledAt:   matchTime,
					HomeTeamID:    pair.Home.ID,
					VisitorTeamID: pair.Visitor.ID,
				}
				matches = append(matches, match)
				matchTime = matchTime.Add(gameDuration)
				matchTime = matchTime.Add(betweenGamesDuration)
			}
			insertMatches(db, tournamentID, matches)
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
func postScore(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("tournamentId")
		log.Println(tournamentID)
		matchID, _ := strconv.Atoi(c.Param("matchId"))
		log.Println(matchID)
		homeTeamGoals, _ := strconv.Atoi(c.FormValue("homeTeamGoals"))
		visitorTeamGoals, _ := strconv.Atoi(c.FormValue("visitorTeamGoals"))
		saveMatchScore(db, matchID, homeTeamGoals, visitorTeamGoals)
		return c.Redirect(http.StatusSeeOther, "/admin/tournaments/"+tournamentID+"/matches")
	}
}
