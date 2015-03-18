#!/bin/bash

usage() {
    echo "Invalid arguments needs a <patch> <directory> "
    exit 1
}

_init() {
    if [ $# -lt 2 ]; then
	usage;
    fi

    if [ ! -f $1 ]; then
	usage;
    fi

    if [ ! -d $2 ]; then
	usage;
    fi

    CODEPATH=$2
    PATCH=$1
    CP="/bin/cp"
    P="/usr/bin/patch"
    B="/usr/bin/basename"
    R="/bin/rm"
}

main () {
    echo ${PWD}
    echo "Copying ${PATCH} to ${CODEPATH}"
    ${CP} -f ${PATCH} ${CODEPATH}
    _pwd=${PWD}

    cd ${CODEPATH}
    patch=$(${B} ${PATCH})
    ${P} -p1 -R < ${patch}
    ${R} -f ${patch}

    cd ${_pwd}
}

_init "$@" && main "$@"
