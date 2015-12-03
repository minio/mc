#!/bin/bash

ME=$(basename "$0")
BASE_DIR=$(dirname "$0")
MINIO_WORK_DIR="$PWD/automated-tests"
TEST_CASE_DIR="$BASE_DIR/tests"

function fatal()
{
    ec=$1
    shift
    if [ "$@" ]; then
        echo "Fatal: $@" >&2
    fi
    exit $ec
}

function error()
{
    echo "Error: $@" >&2
}

function msg()
{
    echo "Info: $@" >&1
}

function go_get()
{
    go get -d -u "$@" && make -C "$GOPATH/src/$@"
    rv=$?

    if [ $rv -ne 0 ]; then
        error "failed to get $@."
        return $rv
    fi
}

function run_minio()
{
    go_get github.com/minio/minio || fatal 1
    [ ! -d "$MINIO_WORK_DIR" ] && mkdir -p "$MINIO_WORK_DIR"
    minio --anonymous server "$MINIO_WORK_DIR" || fatal 1 "failed to run minio server."
}

function install_mc()
{
    go_get github.com/minio/mc || fatal 2
}

function main()
{
    which go >/dev/null || fatal 1 "go executable not found."
    [ "$GOPATH" ] || fatal 1 "GOPATH env not found."

    if [ "$1" == "minio" ]; then
        msg "Running minio server."
        run_minio
    fi

    msg "Running all mc tests."
    for test_script in $TEST_CASE_DIR/*; do
        if [ -x "$test_script" ]; then
            "$test_script" "$@" || error "test $test_script failed."
        fi
    done
}

main "$@"
