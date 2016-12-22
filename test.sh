#! /usr/bin/zsh
echo "Running game $1 times"
cmd="./runGame.sh"; for i in $(seq $1); do $cmd; done | grep "Player #1" | cut -d '#' -f 3 | sort | uniq -c
