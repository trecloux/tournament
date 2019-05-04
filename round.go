package main

type teamPair struct {
	Home    team
	Visitor team
}

func roundRobin(tournamentTeams []team) []teamPair {
	teams := make([]team, len(tournamentTeams))
	copy(teams, tournamentTeams[:len(tournamentTeams)])

	var ghostTeam = team{}
	if len(tournamentTeams)%2 == 1 {
		teams = append(teams, ghostTeam)
	}

	n := len(teams)
	numberOfRounds := n - 1
	gamesPerRound := n / 2

	homeRibbon := make([]team, gamesPerRound)
	copy(homeRibbon, teams[:gamesPerRound])
	awayRibbon := make([]team, gamesPerRound)
	copy(awayRibbon, teams[gamesPerRound:])
	awayRibbon = reverseTeams(awayRibbon)
	fixed := teams[0]

	fixtures := make([]teamPair, 0)
	roundCount := 0
	for roundCount < numberOfRounds {
		roundCount++
		gameCount := 0
		for gameCount < gamesPerRound {
			gameCount++
			teamPair := teamPair{}
			teamPair.Home = homeRibbon[gameCount-1]
			teamPair.Visitor = awayRibbon[gameCount-1]

			if teamPair.Home != ghostTeam && teamPair.Visitor != ghostTeam {
				fixtures = append(fixtures, teamPair)
			}
		}
		// rotate ribbons
		homeToVisitor := homeRibbon[gamesPerRound-1]
		awayToHome := awayRibbon[0]
		homeRibbon = append([]team{fixed}, append([]team{awayToHome}, homeRibbon[1:gamesPerRound-1]...)...)
		awayRibbon = append(awayRibbon[1:], homeToVisitor)
	}
	return fixtures
}

func reverseTeams(input []team) []team {
	if len(input) == 0 {
		return input
	}
	return append(reverseTeams(input[1:]), input[0])
}
