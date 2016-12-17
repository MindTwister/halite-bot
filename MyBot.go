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

func init() {
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

func getOpponentDirections(loc hlt.Location) (d []hlt.Direction) {
	for _, direction := range hlt.CARDINALS {
		site := gameMap.GetSite(loc, direction)
		siteOwner := site.Owner
		if siteOwner != conn.PlayerTag && (siteOwner != neutralOwner || site.Strength < 3) {
			d = append(d, direction)
		}
	}
	return d
}

func getDefeatableNeutralDirections(loc hlt.Location) (d []hlt.Direction) {
	for _, direction := range hlt.CARDINALS {
		site := gameMap.GetSite(loc, direction)
		siteOwner := site.Owner
		if siteOwner != conn.PlayerTag && siteOwner != neutralOwner {
			d = append(d, direction)
		}
	}
	return d
}

func getStrength(loc hlt.Location) int {
	return gameMap.GetSite(loc, hlt.STILL).Strength
}

func getMostValuableNeutralDirections(fromLocation hlt.Location) []hlt.Direction {
	highestValue := -1000
	highValueDirections := make([]hlt.Direction, 0)
	var currentLocation hlt.Location
	for _, direction := range hlt.CARDINALS {
		currentLocation = fromLocation
		log.Printf("Looking towards %v", direction)
		for distance := 1; distance < gameMap.Width/2+1; distance++ {
			currentLocation = gameMap.GetLocation(currentLocation, direction)
			site := gameMap.GetSite(currentLocation, hlt.STILL)
			locationTileOwner := site.Owner

			locationValue := getSiteValue(site) - distance*distance
			if distance > 1 && locationTileOwner == neutralOwner && site.Production > 0 {
				if highestValue < locationValue {
					highestValue = locationValue
					highValueDirections = make([]hlt.Direction, 0)
				}
				if highestValue == locationValue {
					highValueDirections = append(highValueDirections, direction)
				}
				break
			}
			if isNotMe(currentLocation) {
				break
			}
		}
	}
	log.Printf("Most valuable opponent is towards %v", highValueDirections)
	return highValueDirections
}

func getSiteValue(s hlt.Site) int {
	return s.Production*s.Production*s.Production - s.Strength
}

func getClosestEnemy(fromLocation hlt.Location) []hlt.Direction {
	closest := 255
	closestDirections := make([]hlt.Direction, 0)
	var currentLocation hlt.Location
	for _, direction := range hlt.CARDINALS {
		currentLocation = fromLocation
		log.Printf("Looking towards %v", direction)
		for distance := 0; distance < gameMap.Height/2+1; distance++ {
			currentLocation = gameMap.GetLocation(currentLocation, direction)
			site := gameMap.GetSite(currentLocation, hlt.STILL)
			locationTileOwner := site.Owner

			if distance > 0 && locationTileOwner != conn.PlayerTag && locationTileOwner != neutralOwner {
				if distance < closest {
					closest = distance
					closestDirections = make([]hlt.Direction, 0)
				}
				if distance == closest {
					closestDirections = append(closestDirections, direction)
				}
				break
			} else if locationTileOwner != conn.PlayerTag {
				break
			}
		}
	}
	return closestDirections
}

func getWeakestDefeatableNeighbour(fromLocation hlt.Location) (d []hlt.Direction) {
	weakest := 255
	for _, direction := range hlt.CARDINALS {
		site := gameMap.GetSite(fromLocation, direction)
		if site.Strength <= weakest &&
			site.Owner != conn.PlayerTag &&
			shouldAttack(fromLocation, direction) {
			if site.Strength < weakest {
				d = make([]hlt.Direction, 0)
			}
			d = append(d, direction)
		}
	}
	return
}
func getHighestValueNeutralNeighbours(loc hlt.Location) (d []hlt.Direction) {
	mostValue := -10000
	for _, direction := range hlt.CARDINALS {
		site := gameMap.GetSite(loc, direction)
		siteOwner := site.Owner
		siteValue := getSiteValue(site)
		if siteOwner == neutralOwner && siteValue >= mostValue {
			if siteValue > mostValue && shouldAttack(loc, direction) {
				d = make([]hlt.Direction, 0)
				mostValue = siteValue
			}
			if siteValue == mostValue {
				d = append(d, direction)
			}
		}
	}
	return d
}

func getBestDirection(fromLocation hlt.Location) hlt.Direction {
	locationStrength := getStrength(fromLocation)
	if locationStrength < 1 {
		return hlt.STILL
	}
	opponentNeighbours := getOpponentDirections(fromLocation)
	if len(opponentNeighbours) > 0 {
		log.Println("Moving onto opponent")
		return pickRandomDirection(opponentNeighbours)
	}
	defeatableNeighbours := getHighestValueNeutralNeighbours(fromLocation)

	if len(defeatableNeighbours) > 0 {
		log.Println("Conquoring a neutral")
		return pickRandomDirection(defeatableNeighbours)
	}

	site := gameMap.GetSite(fromLocation, hlt.STILL)
	if locationStrength > site.Production*4 || locationStrength > 50 {
		visibleCloseEnemies := getClosestEnemy(fromLocation)
		if len(visibleCloseEnemies) > 0 {
			log.Println("Moving towards enemy")
			return pickRandomDirection(visibleCloseEnemies)
		}
		visibleNeutralDirections := getMostValuableNeutralDirections(fromLocation)
		if len(visibleNeutralDirections) > 0 {
			log.Println("Moving towards neutral")
			return pickRandomDirection(visibleNeutralDirections)
		}
		log.Println("Moving at random")
		return pickRandomDirection(hlt.Directions)
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

func opposite(d hlt.Direction) hlt.Direction {
	return hlt.CARDINALS[(d+1)%4]
}

type lastMoves map[hlt.Location]hlt.Direction

var moveHistory [3]lastMoves

func pruneMoves(ml []hlt.Move) []hlt.Move {
	newMoves := make([]hlt.Move, len(ml))
	for _, l := range moveHistory {
		for _, m := range ml {
			destinationLocation := gameMap.GetLocation(m.Location, m.Direction)
			if lm, ok := l[destinationLocation]; ok && lm == opposite(m.Direction) {
				newMoves = append(newMoves, hlt.Move{m.Location, hlt.STILL})
			} else {
				newMoves = append(newMoves, m)
			}
		}
	}

	newLastMoves := make(lastMoves)
	for _, m := range ml {
		newLastMoves[m.Location] = m.Direction
	}
	for i := 0; i < len(moveHistory)-1; i++ {
		moveHistory[i] = moveHistory[i+1]
	}
	moveHistory[len(moveHistory)-1] = newLastMoves
	return newMoves
}

func main() {
	var wg sync.WaitGroup
	shouldProfile := flag.Bool("profile", false, "Should profiling be done")
	shouldLog := flag.Bool("log", false, "Should logging be done")
	botName := flag.String("name", "StillSortOfRandom", "Bot name")
	flag.Parse()
	conn, gameMap = hlt.NewConnection(*botName)
	neutralOwner = gameMap.GetSite(hlt.NewLocation(0, 0), hlt.STILL).Owner
	f, _ := os.Create("profile.log")
	if *shouldProfile {
		pprof.StartCPUProfile(f)
	}
	if *shouldLog {
		fh, err := os.Create("game.log")
		if err != nil {
			panic(err)
		}
		log.SetOutput(fh)
	} else {
		fh, err := os.Create("/dev/null")
		if err != nil {
			panic(err)
		}
		log.SetOutput(fh)
	}
	count := 0

	lastRoundMoves := 0
	for {
		count++
		preferedRandomDirection = hlt.Direction(rand.Intn(5))
		if *shouldProfile && (count == 300 || lastRoundMoves > 300) {
			pprof.StopCPUProfile()
		}
		lastRoundMoves = 0
		var moves hlt.MoveSet
		gameMap = conn.GetFrame()
		for y := 0; y < gameMap.Height; y++ {
			for x := 0; x < gameMap.Width; x++ {
				loc := hlt.NewLocation(x, y)
				if gameMap.GetSite(loc, hlt.STILL).Owner == conn.PlayerTag {
					lastRoundMoves++
					wg.Add(1)

					go func(loc hlt.Location) {
						moves = append(moves, move(loc))
						wg.Done()
					}(loc)
				}
			}
		}
		wg.Wait()
		moves = pruneMoves(moves)
		log.Printf("Finished with round, sending moves %v", moves)
		conn.SendFrame(moves)
	}
}
