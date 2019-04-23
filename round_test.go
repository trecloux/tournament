package main

import "testing"

func TestComputeRoundRobin(t *testing.T) {
	teams := []team{
		team{ID: 1, Name: "1"},
		team{ID: 2, Name: "2"},
		team{ID: 3, Name: "3"},
		team{ID: 4, Name: "4"},
	}
	pairs := roundRobin(teams)
	if len(pairs) != 6 {
		t.Errorf("Expected %d pairs, got %d.", 6, len(pairs))
	}
	expectPair(t, pairs, 0, 1, 4)
	expectPair(t, pairs, 1, 2, 3)
	expectPair(t, pairs, 2, 1, 3)
	expectPair(t, pairs, 3, 4, 2)
	expectPair(t, pairs, 4, 1, 2)
	expectPair(t, pairs, 5, 3, 4)
}

func expectPair(t *testing.T, pairs []teamPair, index int, expectedHomeID int, expectedAwayID int) {
	if pairs[index].Home.ID != expectedHomeID {
		t.Errorf("Expected pair %d: home team %d, got %d.", index, expectedHomeID, pairs[index].Home.ID)
	}
	if pairs[index].Away.ID != expectedAwayID {
		t.Errorf("Expected pair %d: away team %d, got %d.", index, expectedAwayID, pairs[index].Away.ID)
	}
}
