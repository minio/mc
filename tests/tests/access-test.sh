#!/bin/bash

server="$1"

mc --json access set readonly "$server/testbucket"
# TODO: validate output
