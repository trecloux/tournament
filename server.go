package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
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
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Gzip())
	e.Use(middleware.Recover())

	assetHandler := http.FileServer(rice.MustFindBox("assets").HTTPBox())
	e.GET("/assets/*", echo.WrapHandler(http.StripPrefix("/assets/", assetHandler)))

	e.GET("/", index(db))
	e.GET("/admin", admin(db))
	e.GET("/admin/tournaments/:id/fixtures", adminFixtures(db))
	e.GET("/tournaments/:id/fixtures", getFixtures(db))
	e.GET("/tournaments/:id/ranking", getRanking(db))
	e.POST("/tournaments", generateFixtures(db))
	e.DELETE("/tournaments/:id", removeTournament(db))
	e.POST("/tournaments/:tournamentId/fixtures/:fixtureId/score", postScore(db))

	e.Logger.Fatal(e.Start(":2019"))
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
		return c.Render(http.StatusOK, "admin", echo.Map{"title": "Admin", "tournaments": tournaments})
	}
}
func adminFixtures(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		fixtures := selectFixtures(db, tournamentID)
		return c.Render(http.StatusOK, "fixtures_admin", echo.Map{"title": "Scores", "tournament": tournament, "fixtures": fixtures})
	}
}
func getFixtures(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		fixtures := selectFixtures(db, tournamentID)
		return c.Render(http.StatusOK, "fixtures", echo.Map{"title": "Rencontres", "tournament": tournament, "fixtures": fixtures})
	}
}
func getRanking(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		ranking := selectTournamentRanking(db, tournamentID)
		return c.Render(http.StatusOK, "ranking", echo.Map{"title": "Classeement", "tournament": tournament, "ranking": ranking})
	}
}
func removeTournament(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		deleteTournament(db, tournamentID)
		return c.Redirect(http.StatusSeeOther, "/admin")
	}
}

func generateFixtures(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		startTime, _ := time.Parse("15:04", c.FormValue("start"))
		duration, _ := time.ParseDuration(c.FormValue("duration") + "m")
		tournamentID := c.FormValue("id")
		tournamentName := c.FormValue("name")
		tournament := tournament{
			ID:              tournamentID,
			Name:            tournamentName,
			pointsPerWin:    4,
			pointsPerDraw:   2,
			pointsPerDefeat: 1,
			pointsPerGoal:   0.1,
		}
		insertTournament(db, tournament)

		teamNamesStr := c.FormValue("teams")
		teamNames := strings.Split(teamNamesStr, ",")
		teams := make([]team, 0)
		for _, name := range teamNames {
			teams = append(teams, team{Name: name})
		}
		insertTeams(db, tournamentID, teams)

		// tournament := selectTournament(db, tournamentID)
		teams = selectTournamentTeams(db, tournamentID)
		pairs := roundRobin(teams)
		fixtures := make([]fixture, 0)
		for _, pair := range pairs {
			fixture := fixture{}
			fixture.ScheduledAt = startTime
			fixture.HomeTeamID = pair.Home.ID
			fixture.VisitorTeamID = pair.Visitor.ID
			fixtures = append(fixtures, fixture)
			startTime = startTime.Add(duration)
		}
		insertFixtures(db, tournamentID, fixtures)
		return c.Redirect(http.StatusSeeOther, "/tournaments/"+tournamentID+"/fixtures")
	}
}

func postScore(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("tournamentId")
		log.Println(tournamentID)
		fixtureID, _ := strconv.Atoi(c.Param("fixtureId"))
		log.Println(fixtureID)
		homeTeamGoals, _ := strconv.Atoi(c.FormValue("homeTeamGoals"))
		visitorTeamGoals, _ := strconv.Atoi(c.FormValue("visitorTeamGoals"))
		saveFixtureScore(db, fixtureID, homeTeamGoals, visitorTeamGoals)
		return c.Redirect(http.StatusSeeOther, "/admin/tournaments/"+tournamentID+"/fixtures")
	}
}
