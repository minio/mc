#!/bin/bash

server="$1"

mc --json ls "$server"
# TODO: validate output
