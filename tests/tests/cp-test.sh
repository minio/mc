#!/bin/bash

server="$1"
dir="$2"

minioc --json cp "$dir..." "$server/testbucket"
# TODO: validate output
