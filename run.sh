#!/usr/bin/bash

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <num_iterations>"
  exit 1
fi

port=5000

for _ in $(seq 1 "$1")
do
  ./raftserver.go "127.0.0.1:$port" --disable-logger &
  ((port++))
done