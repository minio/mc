#!/bin/bash

server="$1"

minioc --json ls "$server"
# TODO: validate output
