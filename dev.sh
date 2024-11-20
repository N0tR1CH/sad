#!/opt/homebrew/bin/bash

cleanup() {
    task kill-watch
}

trap cleanup SIGINT

overmind start -f ./Procfile.dev
