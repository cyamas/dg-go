package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

const MpoURL string = "https://statmando.com/rankings/dgpt/mpo"
const FpoURL string = "https://statmando.com/rankings/dgpt/fpo"

func main() {
	teamsFromFile := readAndUnmarshalFile("teams.json")
	league, allMPOPlayers, allFPOPlayers := createLeagueAndAllPlayersSets(teamsFromFile)

	mpoPoints, fpoPoints := getAllPlayerPoints(MpoURL, FpoURL, allMPOPlayers, allFPOPlayers)
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

type PlayersSet map[string]float32

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

func createLeagueAndAllPlayersSets(unmarshaledTeams UnmarshaledTeamsData) ([]Team, PlayersSet, PlayersSet) {
	var league []Team
	allMPOPlayers := make(PlayersSet)
	allFPOPlayers := make(PlayersSet)
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

func addPlayersToSet(playerSet PlayersSet, roster []string) PlayersSet {
	for _, name := range roster {
		playerSet[name] = 0
	}
	return playerSet
}

func getAllPlayerPoints(mpoURL string, fpoURL string, mpoSet, fpoSet PlayersSet) (PlayersSet, PlayersSet) {
	var wg sync.WaitGroup
	wg.Add(2)
	go scrapePlayerPointsByDivision(&mpoSet, mpoURL, &wg)
	go scrapePlayerPointsByDivision(&fpoSet, fpoURL, &wg)
	wg.Wait()

	return mpoSet, fpoSet
}

func scrapePlayerPointsByDivision(players *PlayersSet, url string, wg *sync.WaitGroup) {
	defer wg.Done()
	playersSet := *players

	doc := getHTMLDoc(url)
	tableRowSelector := "#official > tbody > tr"
	nameSelector := "#official > tbody > tr > td.whitespace-nowrap"
	pointsSelector := "#official > tbody > tr > td:nth-child(4)"

	getPlayerPoints := func(_ int, selection *goquery.Selection) {
		rawName := selection.Find(nameSelector).Text()
		name := strings.TrimSpace(rawName)
		if strings.Contains(name, "*") {
			name = name[:len(name)-1]
		}
		_, ok := playersSet[name]
		if ok {
			playersSet[name] = extractPoints(selection, pointsSelector)
		}
	}
	doc.Find(tableRowSelector).Each(getPlayerPoints)
}

func getHTMLDoc(url string) *goquery.Document {
	pageRequest, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer pageRequest.Body.Close()
	if pageRequest.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", pageRequest.StatusCode, pageRequest.Status)
	}
	doc, err := goquery.NewDocumentFromReader(pageRequest.Body)
	if err != nil {
		log.Fatal(err)
	}
	return doc
}

func extractPoints(selection *goquery.Selection, pointsSelector string) float32 {
	rawPointsStr := selection.Find(pointsSelector).Text()
	pointsStr := strings.TrimSpace(rawPointsStr)
	points, err := strconv.ParseFloat(pointsStr, 32)
	if err != nil {
		log.Fatal("Error converting string to Float: ", err)
	}
	return float32(points)
}

func setPlayerPoints(league *[]Team, mpoPoints, fpoPoints PlayersSet) {
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
