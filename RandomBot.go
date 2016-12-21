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
var neutralOwner int = 0
var preferedRandomDirection hlt.Direction

func init() {
}
func hasOnlyFriendlyNeighbours(l hlt.Location) bool {
	for _, d := range hlt.CARDINALS {
		if gameMap.GetSite(l, d).Owner != conn.PlayerTag {
			return false
		}
	}
	return true
}

func isNotMe(loc hlt.Location) bool {
	return gameMap.GetSite(loc, hlt.STILL).Owner != conn.PlayerTag
}

func pickRandomNonReversedDirection(loc hlt.Location, dl []hlt.Direction) hlt.Direction {
	dl = pruneDirections(loc, dl)
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

			locationValue := getSiteValue(gameMap.GetLocation(currentLocation, direction)) - distance
			if (locationTileOwner == neutralOwner) && (site.Production > 0 || site.Strength == 0) {
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

func getSiteValue(l hlt.Location) int {
	value := getStrength(l)
	for _, d := range hlt.CARDINALS {
		s := gameMap.GetSite(l, d)
		if s.Owner != conn.PlayerTag {
			value += s.Production*s.Production - s.Strength
		}
	}
	return value
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
			} else if locationTileOwner != conn.PlayerTag && site.Strength > 5 {
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
		l := gameMap.GetLocation(loc, direction)
		siteOwner := gameMap.GetSite(loc, direction).Owner
		siteValue := getSiteValue(l)
		if siteOwner == neutralOwner && siteValue >= mostValue && shouldAttack(loc, direction) {
			if siteValue > mostValue {
				d = make([]hlt.Direction, 0)
				mostValue = siteValue
			}
			d = append(d, direction)
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
		return pickRandomNonReversedDirection(fromLocation, opponentNeighbours)
	}
	defeatableNeighbours := getHighestValueNeutralNeighbours(fromLocation)

	if len(defeatableNeighbours) > 0 {
		log.Println("Conquoring a neutral")
		return pickRandomNonReversedDirection(fromLocation, defeatableNeighbours)
	}

	site := gameMap.GetSite(fromLocation, hlt.STILL)
	if site.Production*4 < site.Strength || (len(lastMoves) > 15 && locationStrength > 30) {
		visibleCloseEnemies := getClosestEnemy(fromLocation)
		if len(visibleCloseEnemies) > 0 {
			log.Println("Moving towards enemy")
			return pickRandomNonReversedDirection(fromLocation, visibleCloseEnemies)
		}
		visibleNeutralDirections := getMostValuableNeutralDirections(fromLocation)
		if len(visibleNeutralDirections) > 0 {
			log.Println("Moving towards neutral")
			return pickRandomNonReversedDirection(fromLocation, visibleNeutralDirections)
		}
		log.Println("Moving at random")
		if hasOnlyFriendlyNeighbours(fromLocation) {
			return pickRandomNonReversedDirection(fromLocation, hlt.Directions)
		}
	}
	return hlt.STILL
}

func shouldAttack(l hlt.Location, d hlt.Direction) bool {
	return getStrength(l) > getStrength(gameMap.GetLocation(l, d))
}

func move(loc hlt.Location) hlt.Move {
	newMove := hlt.Move{
		Location:  loc,
		Direction: getBestDirection(loc),
	}
	registerMove(newMove)
	return newMove

}

func opposite(d hlt.Direction) hlt.Direction {
	if d == hlt.STILL {
		return hlt.STILL
	}
	return hlt.CARDINALS[(d+1)%4]
}

type moveMap map[hlt.Location]hlt.Direction

var lastMoves moveMap = make(moveMap)
var currentMoves moveMap = make(moveMap)
var rml sync.RWMutex

func registerMove(m hlt.Move) {
	rml.Lock()
	if _, ok := currentMoves[m.Location]; ok {
		lastMoves = currentMoves
		currentMoves = make(moveMap)
	}
	currentMoves[m.Location] = m.Direction
	rml.Unlock()
}

func pruneDirections(loc hlt.Location, directions []hlt.Direction) []hlt.Direction {
	newDirections := make([]hlt.Direction, 0)
	for _, d := range directions {

		destinationLocation := gameMap.GetLocation(loc, d)
		rml.RLock()
		if lm, ok := lastMoves[destinationLocation]; ok && lm == opposite(d) {

		} else {
			newDirections = append(newDirections, d)
		}
		rml.RUnlock()

	}
	if len(newDirections) == 0 {
		newDirections = append(newDirections, hlt.STILL)
	}
	return newDirections
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
	var vmoves sync.Mutex
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
						vmoves.Lock()
						moves = append(moves, move(loc))
						vmoves.Unlock()
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
