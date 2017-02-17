#!/bin/bash

server="$1"

minioc --json access set download "$server/testbucket"
# TODO: validate output
