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
	"time"

	"github.com/gocolly/colly"
)

const MpoURL string = "https://statmando.com/rankings/dgpt/mpo"
const FpoURL string = "https://statmando.com/rankings/dgpt/fpo"

func main() {
	startTime := time.Now()
	teamsFromFile := readFileToLeagueTeams("teams.json")
	allMPOPlayers, allFPOPlayers := collectAllPlayersByDivision(teamsFromFile)

	var wg sync.WaitGroup
	wg.Add(2)
	go scrapeAllPlayerPoints(allMPOPlayers, MpoURL, &wg)
	go scrapeAllPlayerPoints(allFPOPlayers, FpoURL, &wg)
	wg.Wait()

	league := createLeague(teamsFromFile)
	distroPlayersToTeams(league, allMPOPlayers, "mpo")
	distroPlayersToTeams(league, allFPOPlayers, "fpo")

	for i := range league {
		league[i].sortRostersByPoints()
		league[i].sumTopPlayersPoints()
	}
	orderTeamsByPoints(league)
	displayStandings(league)
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	fmt.Println(duration)
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

func readFileToLeagueTeams(file string) LeagueTeams {
	playerLists, err := os.ReadFile(file)
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

func collectAllPlayersByDivision(leagueTeams LeagueTeams) ([]Player, []Player) {
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

func scrapeAllPlayerPoints(players []Player, url string, wg *sync.WaitGroup) {
	defer wg.Done()
	var scraper *colly.Collector = colly.NewCollector()
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

func createLeague(leagueTeams LeagueTeams) []Team {
	var league []Team
	for teamOwner := range leagueTeams {
		var team Team
		team.Owner = teamOwner
		league = append(league, team)
	}
	return league
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
