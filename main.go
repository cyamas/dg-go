package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
)

const MpoURL string = "https://statmando.com/rankings/dgpt/mpo"
const FpoURL string = "https://statmando.com/rankings/dgpt/fpo"

func main() {
	teamsFromFile := readAndUnmarshalFile("teams.json")
	league, allMPOPlayers, allFPOPlayers := splitTeamsAndPlayers(teamsFromFile)

	var wg sync.WaitGroup
	wg.Add(2)
	go scrapeAllPlayerPoints(allMPOPlayers, MpoURL, &wg)
	go scrapeAllPlayerPoints(allFPOPlayers, FpoURL, &wg)
	wg.Wait()

	distroPlayersToTeams(league, allMPOPlayers, "mpo")
	distroPlayersToTeams(league, allFPOPlayers, "fpo")

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

func splitTeamsAndPlayers(unmarshaledTeams UnmarshaledTeamsData) ([]Team, []Player, []Player) {
	var league []Team
	var allMPOPlayers []Player
	var allFPOPlayers []Player
	for owner, rosters := range unmarshaledTeams {
		var team Team
		team.Owner = owner
		league = append(league, team)

		allMPOPlayers = append(allMPOPlayers, addPlayersToAllPlayersList(rosters.MPO, owner)...)
		allFPOPlayers = append(allFPOPlayers, addPlayersToAllPlayersList(rosters.FPO, owner)...)
	}
	return league, allMPOPlayers, allFPOPlayers
}

func addPlayersToAllPlayersList(roster []string, owner string) []Player {
	var allPlayersList []Player
	for _, name := range roster {
		var player Player
		player.Name = name
		player.Owner = owner
		allPlayersList = append(allPlayersList, player)
	}
	return allPlayersList
}

func extractPoints(tableRow *colly.HTMLElement) float32 {
	playerPoints, err := strconv.ParseFloat(tableRow.ChildText("#official > tbody > tr > td:nth-child(4)"), 32)
	if err != nil {
		log.Fatal("Error converting string to Float: ", err)
	}
	return float32(playerPoints)
}

func scrapeAllPlayerPoints(players []Player, url string, wg *sync.WaitGroup) {
	defer wg.Done()
	var scraper *colly.Collector = colly.NewCollector()
	tableSelector := "#official > tbody"

	getPointsFromTable := func(table *colly.HTMLElement) {
		rowSelector := "#official > tbody > tr"
		table.ForEach(rowSelector, func(_ int, tableRow *colly.HTMLElement) {
			nameColumnSelector := "#official > tbody > tr > td.whitespace-nowrap"
			playerName := tableRow.ChildText(nameColumnSelector)
			for i, player := range players {
				if strings.Contains(playerName, player.Name) {
					players[i].Points = extractPoints(tableRow)
				}
			}
		})
	}

	scraper.OnHTML(tableSelector, getPointsFromTable)
	scraper.Visit(url)
}

func distroPlayersToTeams(teams []Team, players []Player, division string) {
	for i, player := range players {
		for j, team := range teams {
			if team.Owner == player.Owner {
				if division == "mpo" {
					teams[j].MPORoster = append(team.MPORoster, players[i])
				} else {
					teams[j].FPORoster = append(team.FPORoster, players[i])
				}
			}
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
