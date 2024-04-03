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

var mpoURL string = "https://statmando.com/rankings/dgpt/mpo"
var fpoURL string = "https://statmando.com/rankings/dgpt/fpo"

func main() {
	teamsFromJSON := readJsonToLeagueTeams()
	allMPOPlayers, allFPOPlayers := createAllPlayersLists(teamsFromJSON)

	scraper := colly.NewCollector()
	scrapeAllPlayerPoints(allMPOPlayers, mpoURL, scraper)
	scrapeAllPlayerPoints(allFPOPlayers, fpoURL, scraper)

	var teams []Team
	for teamOwner := range teamsFromJSON {
		var team Team
		team.Owner = teamOwner
		teams = append(teams, team)
	}
	addPlayersToTeams(teams, allMPOPlayers, "mpo")
	addPlayersToTeams(teams, allFPOPlayers, "fpo")

	for i := range teams {
		teams[i].sortRostersByPoints()
		teams[i].sumTopPlayersPoints()
	}
	orderTeamsByPoints(teams)
	for i, team := range teams {
		fmt.Printf("%v. %v: %v", i+1, team.Owner, team.Points)
	}
}

type LeagueTeams map[string]struct {
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

func readJsonToLeagueTeams() LeagueTeams {
	playerLists, err := os.ReadFile("teams.json")
	if err != nil {
		log.Fatal("Could not read file: ", err)
	}
	var teams LeagueTeams
	err = json.Unmarshal(playerLists, &teams)
	if err != nil {
		log.Fatal("Error unmarshaling JSON: ", err)
	}
	return teams
}

func createAllPlayersLists(leagueTeams LeagueTeams) ([]Player, []Player) {
	var allMPOPlayers []Player
	var allFPOPlayers []Player
	for owner, team := range leagueTeams {
		allMPOPlayers = append(allMPOPlayers, addPlayersToAllPlayersList(team.MPO, owner)...)
		allFPOPlayers = append(allFPOPlayers, addPlayersToAllPlayersList(team.FPO, owner)...)
	}
	return allMPOPlayers, allFPOPlayers
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

func scrapeAllPlayerPoints(players []Player, url string, scraper *colly.Collector) {
	tableSelector := "#official > tbody"
	scraper.OnHTML(tableSelector, func(table *colly.HTMLElement) {
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
	})
	scraper.Visit(url)
}

func addPlayersToTeams(teams []Team, players []Player, division string) {
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
