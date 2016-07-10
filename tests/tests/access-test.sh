#!/bin/bash

server="$1"

mc --json access set download "$server/testbucket"
# TODO: validate output
