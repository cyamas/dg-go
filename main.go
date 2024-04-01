package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
)

type Manager struct {
	Name    string
	Players []Player
	Points  float32
}

type Player struct {
	Name   string
	Points float32
}

var mensURL string = "https://statmando.com/rankings/dgpt/mpo"
var womensURL string = "https://statmando.com/rankings/dgpt/fpo"

var teamShirley Manager

var shirleyMPO = []string{"Isaac Robinson", "James Proctor", "Corey Ellis", "James Conrad", "Väinö Mäkelä", "Paul Krans"}
var shirleyFPO = []string{"Holyn Handley", "Hailey King", "Jessica Weese"}

func main() {
	teamShirley.Name = "Shirley"
	c := colly.NewCollector()

	var wg sync.WaitGroup
	wg.Add(2)

	go scrapeRosteredPlayerPoints(c, mensURL, shirleyMPO, &wg)
	go scrapeRosteredPlayerPoints(c, womensURL, shirleyFPO, &wg)
	wg.Wait()
	fmt.Println(teamShirley)
}

func scrapeRosteredPlayerPoints(c *colly.Collector, URL string, roster []string, wg *sync.WaitGroup) {
	defer wg.Done()
	c.OnHTML("#official > tbody", func(table *colly.HTMLElement) {
		table.ForEach("#official > tbody > tr", func(_ int, row *colly.HTMLElement) {
			rowPlayerName := row.ChildText("#official > tbody > tr > td.whitespace-nowrap")
			for _, rosterName := range roster {
				if strings.Contains(rowPlayerName, rosterName) {
					var player Player
					player.Name = rosterName
					playerPoints := extractPoints(row)
					player.Points = playerPoints
					teamShirley.Players = append(teamShirley.Players, player)
					teamShirley.Points += playerPoints
				}
			}
		})
	})
	c.Visit(URL)
}

func extractPoints(elem *colly.HTMLElement) float32 {
	playerPoints, err := strconv.ParseFloat(elem.ChildText("#official > tbody > tr > td:nth-child(4)"), 32)
	if err != nil {
		log.Fatal("Error converting string to Float: ", err)
	}
	return float32(playerPoints)
}
