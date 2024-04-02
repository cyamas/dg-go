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

type FantasyTeams map[string]struct {
	MPO []string `json:"mpo"`
	FPO []string `json:"fpo"`
}

type Team struct {
	Name    string
	Rosters []Roster
	Points  float32
}

func (team *Team) addRoster(roster Roster) {
	team.Rosters = append(team.Rosters, roster)
}

func collectPlayerPoints(scraper *Scraper, teams []Team) {
	tableSelector := "#official > tbody"
	scraper.Colly.OnHTML(tableSelector, func(table *colly.HTMLElement) {
		scrapePointsForRosteredPlayers(table, teams)
	})
	visitURLs(scraper)
}

func scrapePointsForRosteredPlayers(table *colly.HTMLElement, teams []Team) {
	rowSelector := "#official > tbody > tr"
	table.ForEach(rowSelector, func(_ int, tableRow *colly.HTMLElement) {
		nameColumnSelector := "#official > tbody > tr > td.whitespace-nowrap"
		playerName := tableRow.ChildText(nameColumnSelector)
		for teamIndex, team := range teams {
			for rosterIndex, roster := range team.Rosters {
				for playerIndex, player := range roster.Players {
					rosterName := player.Name
					if strings.Contains(playerName, rosterName) {
						playerPoints := extractPoints(tableRow)
						teams[teamIndex].Rosters[rosterIndex].Players[playerIndex].Points = playerPoints
					}
				}
			}
		}
	})
}

func visitURLs(scraper *Scraper) {
	var wg sync.WaitGroup
	wg.Add(len(scraper.URLs))
	for _, url := range scraper.URLs {
		go func(url string) {
			defer wg.Done()
			scraper.Colly.Visit(url)
		}(url)
	}
	wg.Wait()
}

func (team *Team) sumStarterPoints() {
	for _, roster := range team.Rosters {
		var pointsList []float32
		for _, player := range roster.Players {
			pointsList = append(pointsList, player.Points)
		}
		sort.Slice(pointsList, func(i, j int) bool {
			return pointsList[i] > pointsList[j]
		})
		var sum float32
		if roster.Division == "m" {
			for _, points := range pointsList[:4] {
				sum += points
			}
		} else {
			for _, points := range pointsList[:2] {
				sum += points
			}
		}
		team.Points += sum
	}
}

type Roster struct {
	Division string
	Players  []Player
}

func (roster *Roster) setRoster(division string, players []string) {
	roster.setDivision(division)
	roster.addPlayers(players)
}

func (roster *Roster) setDivision(division string) {
	roster.Division = division
}

func (roster *Roster) addPlayers(playerList []string) {
	for _, name := range playerList {
		var player Player
		player.Name = name
		roster.Players = append(roster.Players, player)
	}
}

type Player struct {
	Name   string
	Points float32
}

type Scraper struct {
	Colly *colly.Collector
	URLs  []string
}

func (scraper *Scraper) createColly() {
	scraper.Colly = colly.NewCollector()
}

var mensURL string = "https://statmando.com/rankings/dgpt/mpo"
var womensURL string = "https://statmando.com/rankings/dgpt/fpo"

func extractPoints(tableRow *colly.HTMLElement) float32 {
	playerPoints, err := strconv.ParseFloat(tableRow.ChildText("#official > tbody > tr > td:nth-child(4)"), 32)
	if err != nil {
		log.Fatal("Error converting string to Float: ", err)
	}
	return float32(playerPoints)
}

func setAllTeams(teams FantasyTeams) []Team {
	var teamList []Team
	for teamName, players := range teams {
		var team Team
		team.Name = teamName

		var rosterMPO Roster
		rosterMPO.setRoster("m", players.MPO)
		team.addRoster(rosterMPO)

		var rosterFPO Roster
		rosterFPO.setRoster("f", players.FPO)
		team.addRoster(rosterFPO)

		teamList = append(teamList, team)
	}
	return teamList
}

func orderByPoints(teams []Team) {
	sort.Slice(teams, func(i, j int) bool {
		return teams[i].Points > teams[j].Points
	})
}

func main() {
	playerLists, err := os.ReadFile("teams.json")
	if err != nil {
		log.Fatal("Could not read file: ", err)
	}
	var teams FantasyTeams
	err = json.Unmarshal(playerLists, &teams)
	if err != nil {
		log.Fatal("Error unmarshaling JSON: ", err)
	}

	var scraper Scraper
	scraper.createColly()
	scraper.URLs = append(scraper.URLs, mensURL)
	scraper.URLs = append(scraper.URLs, womensURL)

	league := setAllTeams(teams)
	collectPlayerPoints(&scraper, league)
	for i := range league {
		league[i].sumStarterPoints()
	}
	orderByPoints(league)
	for i := range league {
		fmt.Printf("%v. %v: %v\n", i+1, league[i].Name, league[i].Points)
	}
}
