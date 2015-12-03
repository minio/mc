#!/bin/bash

server="$1"

mc --json rm --force "$server/testbucket..."
# TODO: validate output
