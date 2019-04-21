package main

import (
	"database/sql"
	"net/http"

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
	e.Use(middleware.Gzip())
	e.Use(middleware.Recover())

	assetHandler := http.FileServer(rice.MustFindBox("assets").HTTPBox())
	e.GET("/assets/*", echo.WrapHandler(http.StripPrefix("/assets/", assetHandler)))

	e.GET("/tournament/:id/fixtures", getFixtures(db))
	e.GET("/tournament/:id/ranking", getRanking(db))

	e.Logger.Fatal(e.Start(":2019"))
}

func getFixtures(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		fixtures := selectFixtures(db, tournamentID)
		return c.Render(http.StatusOK, "fixtures.html", echo.Map{"tournament": tournament, "fixtures": fixtures})
	}
}
func getRanking(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tournamentID := c.Param("id")
		tournament := selectTournament(db, tournamentID)
		ranking := selectTournamentRanking(db, tournamentID)
		return c.Render(http.StatusOK, "ranking.html", echo.Map{"tournament": tournament, "ranking": ranking})
	}
}
