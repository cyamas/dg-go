package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
)

const MpoURL string = "https://statmando.com/rankings/dgpt/mpo"
const FpoURL string = "https://statmando.com/rankings/dgpt/fpo"

func main() {
	teamsFromFile := readAndUnmarshalFile("teams.json")
	league, allMPOPlayers, allFPOPlayers := createLeagueAndAllPlayersSets(teamsFromFile)

	mpoPoints := scrapeAllPlayerPoints(allMPOPlayers, MpoURL)
	fpoPoints := scrapeAllPlayerPoints(allFPOPlayers, FpoURL)
	setPlayerPoints(&league, mpoPoints, fpoPoints)

	for i := range league {
		league[i].sortRostersByPoints()
		league[i].sumTopPlayersPoints()
	}
	orderTeamsByPoints(league)
	displayStandings(league)
}

type UnmarshaledTeamsData map[string]struct {
	MPO []string `json:"mpo"`
	FPO []string `json:"fpo"`
}

type Team struct {
	Owner     string
	MPORoster []Player
	FPORoster []Player
	Points    float32
}

func (team *Team) sortRostersByPoints() {
	sort.Slice(team.MPORoster, func(i, j int) bool {
		return team.MPORoster[i].Points > team.MPORoster[j].Points
	})

	sort.Slice(team.FPORoster, func(i, j int) bool {
		return team.FPORoster[i].Points > team.FPORoster[j].Points
	})
}

func (team *Team) sumTopPlayersPoints() {
	// Take top 4 players from mpo and top 2 players from fpo and sum their points
	for player := range 4 {
		team.Points += team.MPORoster[player].Points
	}
	for player := range 2 {
		team.Points += team.FPORoster[player].Points
	}
}

type Player struct {
	Name   string
	Points float32
	Owner  string
}

func readAndUnmarshalFile(file string) UnmarshaledTeamsData {
	playerLists, err := os.ReadFile(file)
	if err != nil {
		log.Fatal("Could not read file: ", err)
	}
	var teams UnmarshaledTeamsData
	err = json.Unmarshal(playerLists, &teams)
	if err != nil {
		log.Fatal("Error unmarshaling JSON: ", err)
	}
	return teams
}

func createLeagueAndAllPlayersSets(unmarshaledTeams UnmarshaledTeamsData) ([]Team, map[string]float32, map[string]float32) {
	var league []Team
	allMPOPlayers := make(map[string]float32)
	allFPOPlayers := make(map[string]float32)
	for owner, rosters := range unmarshaledTeams {
		var team Team
		team.MPORoster = createPlayerRoster(rosters.MPO, owner)
		team.FPORoster = createPlayerRoster(rosters.FPO, owner)
		team.Owner = owner
		league = append(league, team)

		allMPOPlayers = addPlayersToSet(allMPOPlayers, rosters.MPO)
		allFPOPlayers = addPlayersToSet(allFPOPlayers, rosters.FPO)
	}
	return league, allMPOPlayers, allFPOPlayers
}

func createPlayerRoster(roster []string, owner string) []Player {
	var players []Player
	for _, name := range roster {
		var player Player
		player.Name = name
		player.Owner = owner
		players = append(players, player)
	}
	return players
}

func addPlayersToSet(playerSet map[string]float32, roster []string) map[string]float32 {
	for _, name := range roster {
		playerSet[name] = 0
	}
	return playerSet
}

func extractPoints(tableRow *colly.HTMLElement) float32 {
	playerPoints, err := strconv.ParseFloat(tableRow.ChildText("#official > tbody > tr > td:nth-child(4)"), 32)
	if err != nil {
		log.Fatal("Error converting string to Float: ", err)
	}
	return float32(playerPoints)
}

func scrapeAllPlayerPoints(players map[string]float32, url string) map[string]float32 {
	var scraper *colly.Collector = colly.NewCollector()
	tableSelector := "#official > tbody"

	getPointsFromTable := func(table *colly.HTMLElement) {
		rowSelector := "#official > tbody > tr"
		table.ForEach(rowSelector, func(_ int, tableRow *colly.HTMLElement) {
			nameColumnSelector := "#official > tbody > tr > td.whitespace-nowrap"
			playerName := tableRow.ChildText(nameColumnSelector)
			// DGPT championship qualifiers have * appended to their name in table. Remove *
			if strings.Contains(playerName, "*") {
				playerName = playerName[:len(playerName)-1]
			}
			_, ok := players[playerName]
			if ok {
				players[playerName] = extractPoints(tableRow)
			}
		})
	}

	scraper.OnHTML(tableSelector, getPointsFromTable)
	scraper.Visit(url)
	return players
}

func setPlayerPoints(league *[]Team, mpoPoints, fpoPoints map[string]float32) {
	teams := *league
	for i, team := range teams {
		for m, player := range team.MPORoster {
			teams[i].MPORoster[m].Points = mpoPoints[player.Name]
		}
		for f, player := range team.FPORoster {
			teams[i].FPORoster[f].Points = fpoPoints[player.Name]
		}
	}
}

func orderTeamsByPoints(teams []Team) {
	sort.Slice(teams, func(i, j int) bool {
		return teams[i].Points > teams[j].Points
	})
}

func displayStandings(teams []Team) {
	for i, team := range teams {
		fmt.Printf("%v. %v: %v\n", i+1, team.Owner, team.Points)
	}
}
