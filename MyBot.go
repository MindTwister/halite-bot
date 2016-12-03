package main

import (
	"hlt"
	"log"
	"math/rand"
	"os"
)

var gameMap hlt.GameMap
var conn hlt.Connection
var neutralOwner int

func init() {
	fh, err := os.Create("game.log")
	if err != nil {
		panic(err)
	}
	log.SetOutput(fh)
}

func hasEnemyNeighbour(loc hlt.Location) bool {
	for i := 1; i < 5; i++ {
		direction := hlt.Direction(i)
		siteOwner := gameMap.GetSite(loc, direction).Owner
		if siteOwner != conn.PlayerTag {
			return true
		}
	}
	return false
}

func getStrongestOpponentNeighbours(loc hlt.Location) (d []hlt.Direction) {
	strongest := 0
	for i := 1; i < 5; i++ {
		direction := hlt.Direction(i)
		siteOwner := gameMap.GetSite(loc, direction).Owner
		siteStrength := gameMap.GetSite(loc, direction).Strength
		if siteOwner != conn.PlayerTag && siteOwner != neutralOwner && siteStrength >= strongest {
			if strongest < siteStrength {
				d = make([]hlt.Direction, 0)
				strongest = siteStrength
			}
			d = append(d, direction)
		}
	}
	return d
}

func getWeakestNeutralNeighbours(loc hlt.Location) (d []hlt.Direction) {
	weakest := 255
	for i := 1; i < 5; i++ {
		direction := hlt.Direction(i)
		siteOwner := gameMap.GetSite(loc, direction).Owner
		siteStrength := gameMap.GetSite(loc, direction).Strength
		if siteOwner == neutralOwner && siteStrength <= weakest {
			if weakest > siteStrength {
				d = make([]hlt.Direction, 0)
				weakest = siteStrength
			}
			d = append(d, direction)
		}
	}
	return d
}

func getStrength(loc hlt.Location) int {
	log.Printf("Getting strength for %v", loc)
	return gameMap.GetSite(loc, hlt.STILL).Strength
}

func getDefeatableNeutrals(fromLocation hlt.Location) (d []hlt.Direction) {
	log.Printf("Getting directions for %v", fromLocation)
	directions := getWeakestNeutralNeighbours(fromLocation)
	for _, direction := range directions {
		if shouldAttack(fromLocation, direction) {
			d = append(d, direction)
		}
	}
	log.Printf("Returning directions for %v (%v)", fromLocation, d)
	return d
}

func getClosestCummulativeDefeatable(fromLocation hlt.Location) []hlt.Direction {
	closest := 255
	closestDirections := make([]hlt.Direction, 0)
	var currentLocation hlt.Location
	for _, direction := range hlt.CARDINALS {
		strengthAtDestination := getStrength(fromLocation)
		currentLocation = fromLocation
		log.Printf("Looking towards %v", direction)
		for distance := 0; distance < gameMap.Height; distance++ {
			currentLocation = gameMap.GetLocation(currentLocation, direction)
			locationTileOwner := gameMap.GetSite(currentLocation, hlt.STILL).Owner
			if locationTileOwner != conn.PlayerTag {
				if distance < closest && strengthAtDestination > getStrength(currentLocation) {
					closest = distance
					closestDirections = make([]hlt.Direction, 0)
				}
				if distance == closest && strengthAtDestination > getStrength(currentLocation) {
					closestDirections = append(closestDirections, direction)
				}
				break
			} else {
				strengthAtDestination += getStrength(currentLocation)
			}
		}
	}
	log.Printf("Closest opponent is %v away towards %v", closest, closestDirections)
	return closestDirections
}

func getBestDirection(fromLocation hlt.Location) hlt.Direction {
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
	if getStrength(fromLocation) > 35 {
		cd := getClosestCummulativeDefeatable(fromLocation)
		if len(cd) > 0 && rand.Intn(100) > 25 {
			return cd[rand.Intn(len(cd))]
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
	for {
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
