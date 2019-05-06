package main

type teamPair struct {
	Home    team
	Visitor team
}

func roundRobin(tournamentTeams []team) []teamPair {
	nbTeams := len(tournamentTeams)
	teams := make([]team, nbTeams)
	copy(teams, tournamentTeams[:])

	var ghostTeam = team{}
	if nbTeams%2 == 1 {
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

	matches := make([]teamPair, 0)
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
				matches = append(matches, teamPair)
			}
		}
		// rotate ribbons
		homeToVisitor := homeRibbon[gamesPerRound-1]
		awayToHome := awayRibbon[0]
		homeRibbon = append([]team{fixed}, append([]team{awayToHome}, homeRibbon[1:gamesPerRound-1]...)...)
		awayRibbon = append(awayRibbon[1:], homeToVisitor)
	}
	if nbTeams <= 4 {
		return matches
	}
	orderedMatches := make([]teamPair, 0)
	for i := 0; i < len(matches); i++ {
		current := matches[i]
		if i > 0 && i < len(matches)-1 {
			previous := matches[i-1]
			next := matches[i+1]
			if previous.Home.ID == current.Home.ID || previous.Home.ID == current.Visitor.ID || previous.Visitor.ID == current.Home.ID || previous.Visitor.ID == current.Visitor.ID {
				orderedMatches = append(orderedMatches, next)
				orderedMatches = append(orderedMatches, current)
				i++
			} else {
				orderedMatches = append(orderedMatches, current)
			}

		} else {
			orderedMatches = append(orderedMatches, current)
		}

	}
	return orderedMatches
}

func reverseTeams(input []team) []team {
	if len(input) == 0 {
		return input
	}
	return append(reverseTeams(input[1:]), input[0])
}
