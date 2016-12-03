#!/bin/bash

export GOPATH="$(pwd)"

go build -o mybot MyBot.go

./halite -d "45 45" "./mybot --profile" "go run RandomBot.go"
