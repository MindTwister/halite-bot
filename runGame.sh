#!/bin/bash

export GOPATH="$(pwd)"

go build -o mybot MyBot.go

./halite -d "40 40" "./mybot -profile -name 'MyBot'" "go run RandomBot.go -name 'OpponentBot'"
