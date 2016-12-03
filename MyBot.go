package main

import (
	"hlt"
	"log"
	"math/rand"
	"os"
	"runtime/pprof"
)

var gameMap hlt.GameMap
var conn hlt.Connection
var neutralOwner int

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

func hasEnemyNeighbour(loc hlt.Location) bool {
	for i := 1; i < 5; i++ {
		direction := hlt.Direction(i)
		site := gameMap.GetSite(loc, direction)
		siteOwner := site.Owner
		if siteOwner != conn.PlayerTag && site.Production > 0 {
			return true
		}
	}
	return false
}

func getStrongestOpponentNeighbours(loc hlt.Location) (d []hlt.Direction) {
	strongest := 0
	isTooWeakToIgnore := make([]hlt.Direction, 0)
	for i := 1; i < 5; i++ {
		direction := hlt.Direction(i)
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
	return site.Production*10 - site.Strength
}

func getHighestValueNeutralNeighbours(loc hlt.Location) (d []hlt.Direction) {
	mostValue := 0
	for i := 1; i < 5; i++ {
		direction := hlt.Direction(i)
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

func getClosestStrongestOpponents(fromLocation hlt.Location) []hlt.Direction {
	closest := 255
	closestDirections := make([]hlt.Direction, 0)
	strongest := 0
	var currentLocation hlt.Location
	for _, direction := range hlt.CARDINALS {
		currentLocation = fromLocation
		log.Printf("Looking towards %v", direction)
		for distance := 0; distance < gameMap.Height; distance++ {
			currentLocation = gameMap.GetLocation(currentLocation, direction)
			locationTileOwner := gameMap.GetSite(currentLocation, hlt.STILL).Owner
			locationTileStrength := gameMap.GetSite(currentLocation, hlt.STILL).Strength
			if locationTileOwner != conn.PlayerTag && locationTileOwner != neutralOwner {
				if distance <= closest && strongest >= locationTileStrength {
					if distance < closest || locationTileStrength > strongest {

						closestDirections = make([]hlt.Direction, 0)
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
	return closestDirections
}

func getClosestCummulativeDefeatableNeutral(fromLocation hlt.Location) []hlt.Direction {
	closest := 255
	closestDirections := make([]hlt.Direction, 0)
	var currentLocation hlt.Location
	for _, direction := range hlt.CARDINALS {
		strengthAtDestination := getStrength(fromLocation)
		currentLocation = fromLocation
		log.Printf("Looking towards %v", direction)
		for distance := 0; distance < gameMap.Height; distance++ {
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

func getBestDirection(fromLocation hlt.Location) hlt.Direction {
	locationStrength := getStrength(fromLocation)
	if locationStrength < 5 {
		return hlt.STILL
	}
	so := getStrongestOpponentNeighbours(fromLocation)
	if len(so) > 0 {
		log.Printf("Found opponent to %v", fromLocation)
		return so[rand.Intn(len(so))]
	}
	dn := getDefeatableNeutrals(fromLocation)
	if len(dn) > 0 {
		log.Printf("Found defeatable neutral to %v", fromLocation)
		return dn[rand.Intn(len(dn))]
	}
	if locationStrength > 35 {
		cdo := getClosestStrongestOpponents(fromLocation)
		if len(cdo) > 0 && rand.Intn(100) > 25 {
			return cdo[rand.Intn(len(cdo))]
		}
		cdn := getClosestCummulativeDefeatableNeutral(fromLocation)
		if len(cdn) > 0 && rand.Intn(100) > 25 {
			return cdn[rand.Intn(len(cdn))]
		}
		if rand.Intn(100) > 20 && !hasEnemyNeighbour(fromLocation) {
			return hlt.Direction(1 + rand.Intn(4))
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
	conn, gameMap = hlt.NewConnection("StillSortOfRandom")
	neutralOwner = gameMap.GetSite(hlt.NewLocation(0, 0), hlt.STILL).Owner
	f, _ := os.Create("profile.log")
	pprof.StartCPUProfile(f)
	count := 0
	for {
		count++
		if count == 300 {
			pprof.StopCPUProfile()
		}
		var moves hlt.MoveSet
		gameMap = conn.GetFrame()
		for y := 0; y < gameMap.Height; y++ {
			for x := 0; x < gameMap.Width; x++ {
				loc := hlt.NewLocation(x, y)
				if gameMap.GetSite(loc, hlt.STILL).Owner == conn.PlayerTag {
					moves = append(moves, move(loc))
				}
			}
		}
		log.Printf("Finished with round, sending moves %v", moves)
		conn.SendFrame(moves)
	}
}
