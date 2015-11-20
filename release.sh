#!/bin/bash

echo -n "Making official release binaries.. "
export MC_RELEASE=OFFICIAL
make 1>/dev/null
echo "Binaries built at ${GOPATH}/bin/mc"
