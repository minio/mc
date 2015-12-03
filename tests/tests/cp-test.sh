#!/bin/bash

server="$1"
dir="$2"

mc --json cp "$dir..." "$server/testbucket"
# TODO: validate output
