package main

import (
	"flag"
	"hlt"
	"log"
	"math/rand"
	"os"
	"runtime/pprof"
	"sync"
)

var gameMap hlt.GameMap
var conn hlt.Connection
var neutralOwner int
var preferedRandomDirection hlt.Direction
var canMerge bool

func init() {
	fh, err := os.Create("/dev/null")
	if err != nil {
		panic(err)
	}
	log.SetOutput(fh)
}

func isNotMe(loc hlt.Location) bool {
	return gameMap.GetSite(loc, hlt.STILL).Owner != conn.PlayerTag
}

func pickRandomDirection(dl []hlt.Direction) hlt.Direction {
	return dl[rand.Intn(len(dl))]
}

func hasEnemyNeighbour(loc hlt.Location) bool {
	for _, direction := range hlt.CARDINALS {
		site := gameMap.GetSite(loc, direction)
		siteOwner := site.Owner
		if siteOwner != conn.PlayerTag {
			return true
		}
	}
	return false
}

func getOpponentCount(loc hlt.Location) int {
	count := 0
	for _, direction := range hlt.CARDINALS {
		site := gameMap.GetSite(loc, direction)
		siteOwner := site.Owner
		if siteOwner != conn.PlayerTag && siteOwner != neutralOwner {
			count++
		}
	}
	return count
}

func getStrongestOpponentNeighbours(loc hlt.Location) (d []hlt.Direction) {
	strongest := 0
	isTooWeakToIgnore := make([]hlt.Direction, 0)
	for _, direction := range hlt.CARDINALS {
		siteOwner := gameMap.GetSite(loc, direction).Owner
		siteStrength := gameMap.GetSite(loc, direction).Strength
		if siteStrength < 5 && siteOwner != conn.PlayerTag {
			isTooWeakToIgnore = append(isTooWeakToIgnore, direction)
			continue
		}
		if siteOwner != conn.PlayerTag && siteOwner != neutralOwner && siteStrength >= strongest {
			if strongest < siteStrength {
				d = make([]hlt.Direction, 0)
				strongest = siteStrength
			}
			d = append(d, direction)
		}
	}
	return append(d, isTooWeakToIgnore...)
}

func getLocationValue(loc hlt.Location) int {
	site := gameMap.GetSite(loc, hlt.STILL)
	return site.Production*site.Production - site.Strength
}

func getHighestValueNeutralNeighbours(loc hlt.Location) (d []hlt.Direction) {
	mostValue := -10000
	for _, direction := range hlt.CARDINALS {
		siteOwner := gameMap.GetSite(loc, direction).Owner
		siteValue := getLocationValue(loc)
		if siteOwner == neutralOwner && siteValue >= mostValue {
			if siteValue > mostValue {
				d = make([]hlt.Direction, 0)
				mostValue = siteValue
			}
			d = append(d, direction)
		}
	}
	return d
}

func getStrength(loc hlt.Location) int {
	return gameMap.GetSite(loc, hlt.STILL).Strength
}

func getDefeatableNeutrals(fromLocation hlt.Location) (d []hlt.Direction) {
	log.Printf("Getting directions for %v", fromLocation)
	directions := getHighestValueNeutralNeighbours(fromLocation)
	for _, direction := range directions {
		if shouldAttack(fromLocation, direction) {
			d = append(d, direction)
		}
	}
	log.Printf("Returning directions for %v (%v)", fromLocation, d)
	return d
}

func getClosestOpponents(fromLocation hlt.Location) []hlt.Direction {
	closest := 255
	closestDirections := make([]hlt.Direction, 0)
	var currentLocation hlt.Location
	unmissableDirections := make([]hlt.Direction, 0)
	for _, direction := range hlt.CARDINALS {
		currentLocation = fromLocation
		log.Printf("Looking towards %v", direction)
		for distance := 0; distance < gameMap.Height; distance++ {
			currentLocation = gameMap.GetLocation(currentLocation, direction)
			site := gameMap.GetSite(currentLocation, hlt.STILL)
			locationTileOwner := site.Owner
			if locationTileOwner != conn.PlayerTag && locationTileOwner != neutralOwner {
				if distance <= closest {
					if distance < closest {

						closestDirections = make([]hlt.Direction, 0)
						closest = distance
					}
					closestDirections = append(closestDirections, direction)
				}
			}
			if isNotMe(currentLocation) {
				break
			}
		}
	}
	log.Printf("Closest opponent is %v away towards %v", closest, closestDirections)
	return append(closestDirections, unmissableDirections...)
}

func getClosestCummulativeDefeatableNeutral(fromLocation hlt.Location) []hlt.Direction {
	closest := 255
	closestDirections := make([]hlt.Direction, 0)
	var currentLocation hlt.Location
	for _, direction := range hlt.CARDINALS {
		strengthAtDestination := getStrength(fromLocation)
		currentLocation = fromLocation
		log.Printf("Looking towards %v", direction)
		for distance := 0; distance < 8; distance++ {
			currentLocation = gameMap.GetLocation(currentLocation, direction)
			site := gameMap.GetSite(currentLocation, hlt.STILL)
			locationTileOwner := site.Owner

			locationStrength := getStrength(currentLocation)
			if locationTileOwner == neutralOwner && site.Production > 0 {
				if distance < closest && strengthAtDestination > locationStrength {
					closest = distance
					closestDirections = make([]hlt.Direction, 0)
				}
				if distance == closest && strengthAtDestination > locationStrength {
					closestDirections = append(closestDirections, direction)
				}
				break
			} else {
				strengthAtDestination += locationStrength
			}
			if isNotMe(currentLocation) {
				break
			}
		}
	}
	log.Printf("Closest opponent is %v away towards %v", closest, closestDirections)
	return closestDirections
}

func getClosestEnemy(fromLocation hlt.Location) []hlt.Direction {
	closest := 255
	closestDirections := make([]hlt.Direction, 0)
	var currentLocation hlt.Location
	for _, direction := range hlt.CARDINALS {
		currentLocation = fromLocation
		log.Printf("Looking towards %v", direction)
		for distance := 0; distance < gameMap.Height; distance++ {
			currentLocation = gameMap.GetLocation(currentLocation, direction)
			site := gameMap.GetSite(currentLocation, hlt.STILL)
			locationTileOwner := site.Owner

			if locationTileOwner != conn.PlayerTag {
				if distance < closest {
					closest = distance
					closestDirections = make([]hlt.Direction, 0)
				}
				if distance == closest {
					closestDirections = append(closestDirections, direction)
				}
				break
			}
		}
	}
	return closestDirections
}

func getBestDirection(fromLocation hlt.Location) hlt.Direction {
	locationStrength := getStrength(fromLocation)
	if locationStrength < 5 {
		return hlt.STILL
	}
	if getOpponentCount(fromLocation) > 2 {
		return hlt.STILL
	}
	so := getStrongestOpponentNeighbours(fromLocation)
	if len(so) > 0 {
		log.Printf("Found opponent to %v", fromLocation)
		return pickRandomDirection(so)
	}
	dn := getDefeatableNeutrals(fromLocation)
	if len(dn) > 0 {
		log.Printf("Found defeatable neutral to %v", fromLocation)
		return pickRandomDirection(dn)
	}
	site := gameMap.GetSite(fromLocation, hlt.STILL)
	if getStrength(fromLocation) > site.Production*5 || getStrength(fromLocation) > 50 {
		cdo := getClosestOpponents(fromLocation)
		if len(cdo) > 0 {
			return pickRandomDirection(cdo)
		}
		cdn := getClosestCummulativeDefeatableNeutral(fromLocation)
		if len(cdn) > 0 && canMerge {
			return pickRandomDirection(cdn)
		}
		ce := getClosestEnemy(fromLocation)
		if len(ce) > 0 && !hasEnemyNeighbour(fromLocation) {
			return pickRandomDirection(ce)
		}
		if rand.Intn(100) > 20 && !hasEnemyNeighbour(fromLocation) {
			return hlt.Direction(rand.Intn(2) + 1)
		}
	}
	return hlt.STILL
}

func shouldAttack(l hlt.Location, d hlt.Direction) bool {
	return getStrength(l) > getStrength(gameMap.GetLocation(l, d))
}

func move(loc hlt.Location) hlt.Move {
	return hlt.Move{
		Location:  loc,
		Direction: getBestDirection(loc),
	}

}

func main() {
	var wg sync.WaitGroup
	shouldProfile := flag.Bool("profile", false, "Should profiling be done")
	botName := flag.String("name", "StillSortOfRandom", "Bot name")
	flag.Parse()
	conn, gameMap = hlt.NewConnection(*botName)
	neutralOwner = gameMap.GetSite(hlt.NewLocation(0, 0), hlt.STILL).Owner
	f, _ := os.Create("profile.log")
	if *shouldProfile {
		pprof.StartCPUProfile(f)
	}
	count := 0
	for {
		count++
		preferedRandomDirection = hlt.Direction(rand.Intn(5))
		if *shouldProfile && count == 300 {
			pprof.StopCPUProfile()
		}
		var moves hlt.MoveSet
		gameMap = conn.GetFrame()
		for y := 0; y < gameMap.Height; y++ {
			for x := 0; x < gameMap.Width; x++ {
				loc := hlt.NewLocation(x, y)
				if gameMap.GetSite(loc, hlt.STILL).Owner == conn.PlayerTag {

					canMerge = !canMerge
					wg.Add(1)
					go func(loc hlt.Location) {
						moves = append(moves, move(loc))
						wg.Done()
					}(loc)
				}
			}
		}
		wg.Wait()
		log.Printf("Finished with round, sending moves %v", moves)
		conn.SendFrame(moves)
	}
}
