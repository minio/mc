#!/bin/bash

server="$1"

minioc --json rm --force "$server/testbucket..."
# TODO: validate output
