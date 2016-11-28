package main

import (
	"hlt"
	"log"
	"os"

	"math/rand"
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

func hasEnemyNeighBourgh(loc hlt.Location) bool {
	for i := 1; i < 5; i++ {
		direction := hlt.Direction(i)
		if gameMap.GetSite(loc, direction).Owner != conn.PlayerTag {
			return true
		}
	}
	return false
}

func getStrength(loc hlt.Location) int {
	log.Printf("Getting strength for %v", loc)
	return gameMap.GetSite(loc, hlt.STILL).Strength
}

func getDefeatableOpponents(fromLocation hlt.Location) []hlt.Direction {
	log.Printf("Getting directions for %v", fromLocation)
	directions := make([]hlt.Direction, 0)
	for i := 1; i < 5; i++ {
		direction := hlt.Direction(i)
		if gameMap.GetSite(fromLocation, direction).Owner != conn.PlayerTag {
			if shouldAttack(fromLocation, direction) {
				directions = append(directions, direction)
			}
		}
	}
	log.Printf("Returning directions for %v (%v)", fromLocation, directions)
	return directions
}

func getClosestEnemyDirection(fromLocation hlt.Location) hlt.Direction {
	closest := 255
	closestDirection := hlt.Direction(rand.Intn(5))
	var currentLocation hlt.Location
	closestDirectionIsOpponent := false
	for _, direction := range hlt.CARDINALS {
		currentLocation = fromLocation
		log.Printf("Looking towards %v", direction)
		for distance := 0; distance < 30; distance++ {
			currentLocation = gameMap.GetLocation(currentLocation, direction)
			locationTileOwner := gameMap.GetSite(currentLocation, hlt.STILL).Owner
			isOpponent := locationTileOwner != neutralOwner
			if locationTileOwner != conn.PlayerTag {
				if isOpponent && !closestDirectionIsOpponent {
					closest = distance
					closestDirection = direction
					closestDirectionIsOpponent = true
				} else if isOpponent && distance < closest {
					closest = distance
					closestDirection = direction
				} else if distance < closest {
					closest = distance
					closestDirection = direction
				}
				break
			}
		}
	}
	log.Printf("Closest enemy is %v away towards %v", closest, closestDirection)
	return closestDirection
}

func getWeakestDirection(fromLocation hlt.Location, directions []hlt.Direction) hlt.Direction {
	weakest := 255
	var weakestDirection hlt.Direction
	for _, direction := range directions {
		if gameMap.GetSite(fromLocation, direction).Strength < weakest {
			weakest = gameMap.GetSite(fromLocation, direction).Strength
			weakestDirection = direction
		}
	}
	return weakestDirection
}

func getFriendlyNeighbours(fromLocation hlt.Location) hlt.Direction {
	weakest := 255
	var weakestDirection hlt.Direction = hlt.STILL
	for _, direction := range hlt.CARDINALS {
		if gameMap.GetSite(fromLocation, direction).Strength < weakest && gameMap.GetSite(fromLocation, direction).Owner == conn.PlayerTag {
			weakest = gameMap.GetSite(fromLocation, direction).Strength
			weakestDirection = direction
		}
	}
	return weakestDirection
}

func getBestDirection(fromLocation hlt.Location) hlt.Direction {
	log.Printf("Finding best direction for %v", fromLocation)
	if !hasEnemyNeighBourgh(fromLocation) {
		if getStrength(fromLocation) < 40 {
			if rand.Intn(100) < 10 {
				return getFriendlyNeighbours(fromLocation)
			}
			log.Printf("Recommendation, standing still for %v", fromLocation)
			return hlt.STILL
		}
		return getClosestEnemyDirection(fromLocation)
	}
	availableDirections := getDefeatableOpponents(fromLocation)
	if len(availableDirections) > 0 {
		log.Printf("Attacking a random direction %v", fromLocation)
		return getWeakestDirection(fromLocation, availableDirections)
	}
	log.Printf("Recommendation, standing still for %v", fromLocation)
	return hlt.STILL
}

func shouldAttack(myLocation hlt.Location, direction hlt.Direction) bool {
	if gameMap.GetSite(myLocation, direction).Owner != neutralOwner {
		log.Printf("Not a neutral neighbour, attack!")
		return true
	}
	if getStrength(myLocation) > 225 {
		return true
	}
	return getStrength(myLocation) > gameMap.GetSite(myLocation, direction).Strength
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
