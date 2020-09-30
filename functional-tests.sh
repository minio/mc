#!/bin/bash
#
# MinIO Client (C) 2017-2020 MinIO, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

################################################################################
#
# This script is usable by mc functional tests, mint tests and MinIO verification
# tests.
#
# * As mc functional tests, just run this script.  It uses mc executable binary
#   in current working directory or in the path.  The tests uses play.min.io
#   as MinIO server.
#
# * For other, call this script with environment variables MINT_MODE,
#   MINT_DATA_DIR, SERVER_ENDPOINT, ACCESS_KEY, SECRET_KEY and ENABLE_HTTPS. It
#   uses mc executable binary in current working directory and uses given MinIO
#   server to run tests. MINT_MODE is set by mint to specify what category of
#   tests to run.
#
################################################################################

# Force bytewise sorting for CLI tools
LANG=C

if [ -n "$MINT_MODE" ]; then
    if [ -z "${MINT_DATA_DIR+x}" ]; then
        echo "MINT_DATA_DIR not defined"
        exit 1
    fi
    if [ -z "${SERVER_ENDPOINT+x}" ]; then
        echo "SERVER_ENDPOINT not defined"
        exit 1
    fi
    if [ -z "${ACCESS_KEY+x}" ]; then
        echo "ACCESS_KEY not defined"
        exit 1
    fi
    if [ -z "${SECRET_KEY+x}" ]; then
        echo "SECRET_KEY not defined"
        exit 1
    fi
fi

if [ -z "${SERVER_ENDPOINT+x}" ]; then
    SERVER_ENDPOINT="play.min.io"
    ACCESS_KEY="Q3AM3UQ867SPQQA43P2F"
    SECRET_KEY="zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG"
    ENABLE_HTTPS=1
fi

WORK_DIR="$PWD"
DATA_DIR="$MINT_DATA_DIR"
if [ -z "$MINT_MODE" ]; then
    WORK_DIR="$PWD/.run-$RANDOM"
    DATA_DIR="$WORK_DIR/data"
fi

FILE_0_B="$DATA_DIR/datafile-0-b"
FILE_1_MB="$DATA_DIR/datafile-1-MB"
FILE_65_MB="$DATA_DIR/datafile-65-MB"
declare FILE_0_B_MD5SUM
declare FILE_1_MB_MD5SUM
declare FILE_65_MB_MD5SUM

ENDPOINT="https://$SERVER_ENDPOINT"
if [ "$ENABLE_HTTPS" != "1" ]; then
    ENDPOINT="http://$SERVER_ENDPOINT"
fi

SERVER_ALIAS="myminio"
SERVER_ALIAS_TLS="myminio-ssl"

BUCKET_NAME="mc-test-bucket-$RANDOM"
WATCH_OUT_FILE="$WORK_DIR/watch.out-$RANDOM"

MC_CONFIG_DIR="/tmp/.mc-$RANDOM"
MC="$PWD/mc"
declare -a MC_CMD

function get_md5sum()
{
    filename="$1"
    out=$(md5sum "$filename" 2>/dev/null)
    rv=$?
    if [ "$rv" -eq 0 ]; then
        echo $(awk '{ print $1 }' <<< "$out")
    fi

    return "$rv"
}

function get_time()
{
    date +%s%N
}

function get_duration()
{
    start_time=$1
    end_time=$(get_time)

    echo $(( (end_time - start_time) / 1000000 ))
}

function log_success()
{
    if [ -n "$MINT_MODE" ]; then
        printf '{"name": "mc", "duration": "%d", "function": "%s", "status": "PASS"}\n' "$(get_duration "$1")" "$2"
    fi
}

function show()
{
    if [ -z "$MINT_MODE" ]; then
        func_name="$1"
        echo "Running $func_name()"
    fi
}

function show_on_success()
{
    rv="$1"
    shift

    if [ "$rv" -eq 0 ]; then
        echo "$@"
    fi

    return "$rv"
}

function show_on_failure()
{
    rv="$1"
    shift

    if [ "$rv" -ne 0 ]; then
        echo "$@"
    fi

    return "$rv"
}

function assert()
{
    expected_rv="$1"
    shift
    start_time="$1"
    shift
    func_name="$1"
    shift

    err=$("$@")
    rv=$?
    if [ "$rv" -ne "$expected_rv" ]; then
        if [ -n "$MINT_MODE" ]; then
            err=$(printf "$err" | python -c 'import sys,json; print(json.dumps(sys.stdin.read()))')
            ## err is already JSON string, no need to double quote
            printf '{"name": "mc", "duration": "%d", "function": "%s", "status": "FAIL", "error": %s}\n' "$(get_duration "$start_time")" "$func_name" "$err"
        else
            echo "mc: $func_name: $err"
        fi
        
        if [ "$rv" -eq 0 ]; then
            exit 1
        fi

        exit "$rv"
    fi

    return 0
}

function assert_success() {
    assert 0 "$@"
}

function assert_failure() {
    assert 1 "$@"
}

function mc_cmd()
{
    cmd=( "${MC_CMD[@]}" "$@" )
    err_file="$WORK_DIR/cmd.out.$RANDOM"

    "${cmd[@]}" >"$err_file" 2>&1
    rv=$?
    if [ "$rv" -ne 0 ]; then
        printf '%q ' "${cmd[@]}"
        echo " >>> "
        cat "$err_file"
    fi

    rm -f "$err_file"
    return "$rv"
}

function check_md5sum()
{
    expected_checksum="$1"
    shift
    filename="$@"

    checksum="$(get_md5sum "$filename")"
    rv=$?
    if [ "$rv" -ne 0 ]; then
        echo "unable to get md5sum for $filename"
        return "$rv"
    fi

    if [ "$checksum" != "$expected_checksum" ]; then
        echo "$filename: md5sum mismatch"
        return 1
    fi

    return 0
}

function test_make_bucket()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    bucket_name="mc-test-bucket-$RANDOM"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${bucket_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rb "${SERVER_ALIAS}/${bucket_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_make_bucket_error() {
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    bucket_name="MC-test%bucket%$RANDOM"
    assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${bucket_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function setup()
{
    start_time=$(get_time)
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${BUCKET_NAME}"
}

function teardown()
{
    start_time=$(get_time)
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rb --force "${SERVER_ALIAS}/${BUCKET_NAME}"
}

function test_put_object()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_put_object_error()
{
    show "${FUNCNAME[0]}"
    start_time=$(get_time)

    object_long_name=$(printf "mc-test-object-%01100d" 1)
    assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_long_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_put_object_multipart()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_65_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_put_object_0byte()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_0_B}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_0_B_MD5SUM" "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

## Test mc cp command with storage-class flag set
function test_put_object_with_storage_class()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --storage-class REDUCED_REDUNDANCY "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

## Test mc cp command with storage-class flag set to incorrect value
function test_put_object_with_storage_class_error()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd cp --storage-class REDUCED "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

## Test mc cp command with valid metadata string
function test_put_object_with_metadata()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --attr key1=val1\;key2=val2 "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    diff -bB <(echo "val1")  <("${MC_CMD[@]}"   --json stat "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"  |  jq -r '.metadata."X-Amz-Meta-Key1"')  >/dev/null 2>&1
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to put object with metadata"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    log_success "$start_time" "${FUNCNAME[0]}"

}

function test_get_object()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_get_object_multipart()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_65_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_65_MB_MD5SUM" "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_presigned_post_policy_error()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"

    out=$("${MC_CMD[@]}" --json share upload "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}")
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to get presigned post policy and put object url"

    # Support IPv6 address, IPv6 is specifed as [host]:9000 form, we should
    # replace '['']' with their escaped values as '\[' '\]'.
    #
    # Without escaping '['']', 'sed' command interprets them as expressions
    # which fails our requirement of replacing $endpoint/$bucket URLs in the
    # subsequent operations.
    endpoint=$(echo "$ENDPOINT" | sed 's|[][]|\\&|g')

    # Extract share field of json output, and append object name to the URL
    upload=$(echo "$out" | jq -r .share | sed "s|<FILE>|$FILE_1_MB|g" | sed "s|curl|curl -sSg|g" | sed "s|${endpoint}/${BUCKET_NAME}/|${endpoint}/${BUCKET_NAME}/${object_name}|g")

    # In case of virtual host style URL path, the previous replace would have failed.
    # One of the following two commands will append the object name in that scenario.
    upload=$(echo "$upload" | sed "s|http://${BUCKET_NAME}.${SERVER_ENDPOINT}/|http://${BUCKET_NAME}.${SERVER_ENDPOINT}/${object_name}|g")
    upload=$(echo "$upload" | sed "s|https://${BUCKET_NAME}.${SERVER_ENDPOINT}/|https://${BUCKET_NAME}.${SERVER_ENDPOINT}/${object_name}|g")

    ret=$($upload 2>&1 | grep -oP '(?<=Code>)[^<]+')
    # Check if the command execution failed.
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unknown failure in upload of $FILE_1_MB using presigned post policy"
    if [ -z "$ret" ]; then

    # Check if the upload succeeded. We expect it to fail.
    assert_failure "$start_time" "${FUNCNAME[0]}" show_on_success 0 "upload of $FILE_1_MB using presigned post policy should have failed"
    fi

    if [ "$ret" != "MethodNotAllowed" ]; then
    assert_failure "$start_time" "${FUNCNAME[0]}" show_on_success 0 "upload of $FILE_1_MB using presigned post policy should have failed with MethodNotAllowed error, instead failed with $ret error"
    fi
    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_presigned_put_object()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"

    out=$("${MC_CMD[@]}" --json share upload "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}")
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to get presigned put object url"
    upload=$(echo "$out" | jq -r .share | sed "s|<FILE>|$FILE_1_MB|g" | sed "s|curl|curl -sSg|g")
    $upload >/dev/null 2>&1
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to upload $FILE_1_MB presigned put object url"

    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_presigned_get_object()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    out=$("${MC_CMD[@]}" --json share download "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}")
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to get presigned get object url"
    download_url=$(echo "$out" | jq -r .share)
    curl -sSg --output "${object_name}.downloaded" -X GET "$download_url"
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to download $download_url"

    assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_cat_object()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    "${MC_CMD[@]}" cat "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" > "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to download object using 'mc cat'"
    assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_cat_stdin()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    echo "testcontent" | "${MC_CMD[@]}" cat > stdin.output
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to redirect stdin to stdout using 'mc cat'"
    assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "42ed9fb3563d8e9c7bb522be443033f4" stdin.output
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm stdin.output

    log_success "$start_time" "${FUNCNAME[0]}"
}


function test_mirror_list_objects()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    bucket_name="mc-test-bucket-$RANDOM"
    object_name="mc-test-object-$RANDOM"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${bucket_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mirror "$DATA_DIR" "${SERVER_ALIAS}/${bucket_name}"

    diff -bB <(ls "$DATA_DIR") <("${MC_CMD[@]}" --json ls "${SERVER_ALIAS}/${bucket_name}/" | jq -r .key) >/dev/null 2>&1
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "mirror and list differs"

    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rb --force "${SERVER_ALIAS}/${bucket_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

## Tests mc mirror command with --storage-class flag set
function test_mirror_list_objects_storage_class()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    bucket_name="mc-test-bucket-$RANDOM"
    object_name="mc-test-object-$RANDOM"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${bucket_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mirror --storage-class REDUCED_REDUNDANCY "$DATA_DIR" "${SERVER_ALIAS}/${bucket_name}"

    diff -bB <(ls "$DATA_DIR") <("${MC_CMD[@]}" --json ls "${SERVER_ALIAS}/${bucket_name}/" | jq -r .key) >/dev/null 2>&1
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "mirror and list differs"

    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rb --force "${SERVER_ALIAS}/${bucket_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

## Tests find command with --older-than set to 1day, should be empty.
function test_find_empty() {
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    bucket_name="mc-test-bucket-$RANDOM"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${bucket_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mirror "$DATA_DIR" "${SERVER_ALIAS}/${bucket_name}"

    # find --older-than 1 day should be empty, so we compare with empty string.
    diff -bB <(echo "") <("${MC_CMD[@]}" find "${SERVER_ALIAS}/${bucket_name}" --json --older-than 1d | jq -r .key | sed "s/${SERVER_ALIAS}\/${bucket_name}\///g") >/dev/null 2>&1
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "mirror and list differs"

    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rb --force "${SERVER_ALIAS}/${bucket_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

## Tests find command, should list.
function test_find() {
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    bucket_name="mc-test-bucket-$RANDOM"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${bucket_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mirror "$DATA_DIR" "${SERVER_ALIAS}/${bucket_name}"

    diff -bB <(ls "$DATA_DIR") <("${MC_CMD[@]}" --json find "${SERVER_ALIAS}/${bucket_name}/" | jq -r .key | sed "s/${SERVER_ALIAS}\/${bucket_name}\///g") >/dev/null 2>&1
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "mirror and list differs"

    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rb --force "${SERVER_ALIAS}/${bucket_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_watch_object()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    bucket_name="mc-test-bucket-$RANDOM"
    object_name="mc-test-object-$RANDOM"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${bucket_name}"

    # start a process to watch on bucket
    "${MC_CMD[@]}" --json watch "${SERVER_ALIAS}/${bucket_name}" > "$WATCH_OUT_FILE" &
    watch_cmd_pid=$!
    sleep 1

    ( assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${bucket_name}/${object_name}" )
    rv=$?
    if [ "$rv" -ne 0 ]; then
        kill "$watch_cmd_pid"
        exit "$rv"
    fi

    sleep 1
    if ! jq -r .events.type "$WATCH_OUT_FILE" | grep -qi ObjectCreated; then
        kill "$watch_cmd_pid"
        assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure 1 "ObjectCreated event not found"
    fi

    ( assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${bucket_name}/${object_name}" )
    rv=$?
    if [ "$rv" -ne 0 ]; then
        kill "$watch_cmd_pid"
        exit "$rv"
    fi

    sleep 1
    if ! jq -r .events.type "$WATCH_OUT_FILE" | grep -qi ObjectRemoved; then
        kill "$watch_cmd_pid"
        assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure 1 "ObjectRemoved event not found"
    fi

    kill "$watch_cmd_pid"

    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rb --force "${SERVER_ALIAS}/${bucket_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}


function test_config_host_add()
{
    show "${FUNCNAME[0]}"
    start_time=$(get_time)

    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd alias set "${SERVER_ALIAS}1" "$ENDPOINT" "$ACCESS_KEY" "$SECRET_KEY"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd alias list "${SERVER_ALIAS}1"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_config_host_add_error()
{
    show "${FUNCNAME[0]}"
    start_time=$(get_time)

    out=$("${MC_CMD[@]}" --json alias set "${SERVER_ALIAS}1" "$ENDPOINT" "$ACCESS_KEY" "invalid-secret")
    assert_failure "$start_time" "${FUNCNAME[0]}" show_on_success $? "adding host should fail"
    got_code=$(echo "$out" | jq -r .error.cause.error.Code)
    if [ "${got_code}" != "SignatureDoesNotMatch" ]; then
        assert_failure "$start_time" "${FUNCNAME[0]}" show_on_failure 1 "incorrect error code ${got_code} returned by server"
    fi

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_put_object_with_sse()
{
    show "${FUNCNAME[0]}"
    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}=32byteslongsecretkeymustbegiven1"
    # put encrypted object; then delete with correct secret key
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --encrypt-key "${cli_flag}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm --encrypt-key "${cli_flag}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_put_object_with_encoded_sse()
{
    show "${FUNCNAME[0]}"
    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}=MzJieXRlc2xvbmdzZWNyZWFiY2RlZmcJZ2l2ZW5uMjE="
    # put encrypted object; then delete with correct secret key
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --encrypt-key "${cli_flag}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm --encrypt-key "${cli_flag}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_put_object_with_sse_error()
{
    show "${FUNCNAME[0]}"
    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}=32byteslongsecretkeymustbegiven"
    # put object with invalid encryption key; should fail
    assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd cp --encrypt-key "${cli_flag}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_cat_object_with_sse()
{
    show "${FUNCNAME[0]}"
    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}=32byteslongsecretkeymustbegiven1"
    # put encrypted object; then cat object correct secret key
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp  --encrypt-key "${cli_flag}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cat --encrypt-key "${cli_flag}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_cat_object_with_sse_error()
{
    show "${FUNCNAME[0]}"
    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}=32byteslongsecretkeymustbegiven1"
    # put encrypted object; then cat object with no secret key
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp  --encrypt-key "${cli_flag}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd cat  "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    log_success "$start_time" "${FUNCNAME[0]}"
}

# Test "mc cp -a" command to see if it preserves file system attributes
function test_copy_object_preserve_filesystem_attr()
{
    show "${FUNCNAME[0]}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp -a "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    diff -bB <("${MC_CMD[@]}"   --json stat "${FILE_1_MB}"  |  jq -r '.metadata."X-Amz-Meta-Mc-Attrs"')  >/dev/null 2>&1 <("${MC_CMD[@]}"   --json stat "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"  |  jq -r '.metadata."X-Amz-Meta-Mc-Attrs"')  >/dev/null 2>&1
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to put object with file system attribute"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_copy_object_with_sse_rewrite()
{
    # test server side copy and remove operation - target is unencrypted while source is encrypted
    show "${FUNCNAME[0]}"
    start_time=$(get_time)
    prefix="prefix"
    object_name="mc-test-object-$RANDOM"

    cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}=32byteslongsecretkeymustbegiven1"
    # create encrypted object on server
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --encrypt-key "${cli_flag}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}/${object_name}"
    # now do a server side copy and store it unencrypted.
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --encrypt-key "${cli_flag}" "${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}/${object_name}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    # cat the unencrypted destination object. should return data without any error
    "${MC_CMD[@]}" cat "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" >"${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to download object using 'mc cat'"
    assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded"
    # mc rm on with multi-object delete, deletes encrypted object without encryption key.
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm --encrypt-key "${cli_flag}" "${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}/${object_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_copy_object_with_sse_dest()
{
    # test server side copy and remove operation - target is encrypted with different key
    show "${FUNCNAME[0]}"
    start_time=$(get_time)
    prefix="prefix"
    object_name="mc-test-object-$RANDOM"

    cli_flag1="${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}=32byteslongsecretkeymustbegiven1"
    cli_flag2="${SERVER_ALIAS}/${BUCKET_NAME}=32byteslongsecretkeymustbegiven2"

    # create encrypted object on server
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --encrypt-key "${cli_flag1}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}/${object_name}"
    # now do a server side copy and store it eith different encryption key.
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --encrypt-key "${cli_flag1},${cli_flag2}"  "${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}/${object_name}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    # cat the destination object with the new key. should return data without any error
    "${MC_CMD[@]}" cat --encrypt-key "${cli_flag2}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" > "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to download object using 'mc cat'"
    assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded"
    # mc rm on src object with first encryption key should pass
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm --encrypt-key "${cli_flag1}" "${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}/${object_name}"
    # mc rm on encrypted destination object with second encryption key should pass
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm  --encrypt-key "${cli_flag2}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_sse_key_rotation()
{
    # test server side copy and remove operation - target is encrypted with different key
    show "${FUNCNAME[0]}"
    start_time=$(get_time)
    prefix="prefix"
    object_name="mc-test-object-$RANDOM"
    old_key="32byteslongsecretkeymustbegiven1"
    new_key="32byteslongsecretkeymustbegiven2"
    cli_flag1="${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}=${old_key}"
    cli_flag2="${SERVER_ALIAS_TLS}/${BUCKET_NAME}=${new_key}"

    # create encrypted object on server
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --encrypt-key "${cli_flag1}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}/${object_name}"
    # now do a server side copy on same object and do a key rotation
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --encrypt-key "${cli_flag1},${cli_flag2}"  "${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}/${object_name}" "${SERVER_ALIAS_TLS}/${BUCKET_NAME}/${object_name}"
    # cat the object with the new key. should return data without any error
    "${MC_CMD[@]}" cat --encrypt-key "${cli_flag2}" "${SERVER_ALIAS_TLS}/${BUCKET_NAME}/${object_name}" > "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to download object using 'mc cat'"
    assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded"
    # mc rm on encrypted object with succeed anyways, without encrypted keys.
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS_TLS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_mirror_with_sse()
{
    # test if mirror operation works with encrypted objects
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    bucket_name="mc-test-bucket-$RANDOM"
    cli_flag="${SERVER_ALIAS}/${bucket_name}=32byteslongsecretkeymustbegiven1"

    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${bucket_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mirror --encrypt-key "${cli_flag}" "$DATA_DIR" "${SERVER_ALIAS}/${bucket_name}"
    diff -bB <(ls "$DATA_DIR") <("${MC_CMD[@]}" --json ls "${SERVER_ALIAS}/${bucket_name}/" | jq -r .key) >/dev/null 2>&1
    assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "mirror and list differs"
    # Remove the test bucket with its contents
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rb --force "${SERVER_ALIAS}/${bucket_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_rm_object_with_sse()
{
    show "${FUNCNAME[0]}"

    # test whether remove fails for encrypted object if secret key not provided.
    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}=32byteslongsecretkeymustbegiven1"

    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --encrypt-key "${cli_flag}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    # rm will not fail even if the encryption keys are not provided, since mc rm uses multi-object delete.
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_get_object_with_sse()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}=32byteslongsecretkeymustbegiven1"

    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --encrypt-key "${cli_flag}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --encrypt-key "${cli_flag}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${object_name}.downloaded"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm --encrypt-key "${cli_flag}" "${object_name}.downloaded" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function test_put_object_multipart_sse()
{
    show "${FUNCNAME[0]}"

    start_time=$(get_time)
    object_name="mc-test-object-$RANDOM"
    cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}=32byteslongsecretkeymustbegiven1"

    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --encrypt-key "${cli_flag}" "${FILE_65_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm --encrypt-key "${cli_flag}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

    log_success "$start_time" "${FUNCNAME[0]}"
}

function run_test()
{
    test_make_bucket
    test_make_bucket_error

    setup
    test_put_object
    test_put_object_error
    test_put_object_0byte
    test_put_object_with_storage_class
    test_put_object_with_storage_class_error
    test_put_object_with_metadata
    test_put_object_multipart
    test_get_object
    test_get_object_multipart
    test_presigned_post_policy_error
    test_presigned_put_object
    test_presigned_get_object
    test_cat_object
    test_cat_stdin
    test_mirror_list_objects
    test_mirror_list_objects_storage_class
    test_copy_object_preserve_filesystem_attr
    test_find
    test_find_empty
    if [ -z "$MINT_MODE" ]; then
        test_watch_object
    fi

    if [ "$ENABLE_HTTPS" == "1" ]; then
        test_put_object_with_sse
        test_put_object_with_encoded_sse
        test_put_object_with_sse_error
        test_put_object_multipart_sse
        test_get_object_with_sse
        test_cat_object_with_sse
        test_cat_object_with_sse_error
        test_copy_object_with_sse_rewrite
        test_copy_object_with_sse_dest
        test_sse_key_rotation
        test_mirror_with_sse
        test_rm_object_with_sse
    fi

    test_config_host_add
    test_config_host_add_error

    teardown
}

function __init__()
{
    set -e
    # For Mint, setup is already done.  For others, setup the environment
    if [ -z "$MINT_MODE" ]; then
        mkdir -p "$WORK_DIR"
        mkdir -p "$DATA_DIR"

        # If mc executable binary is not available in current directory, use it in the path.
        if [ ! -x "$MC" ]; then
            if ! MC=$(which mc 2>/dev/null); then
                echo "'mc' executable binary not found in current directory and in path"
                exit 1
            fi
        fi
    fi

    if [ ! -x "$MC" ]; then
        echo "$MC executable binary not found"
        exit 1
    fi

    mkdir -p "$MC_CONFIG_DIR"
    MC_CMD=( "${MC}" --config-dir "$MC_CONFIG_DIR" --quiet --no-color )

    if [ ! -e "$FILE_0_B" ]; then
        base64 /dev/urandom | head -c 0 >"$FILE_0_B"
    fi

    if [ ! -e "$FILE_1_MB" ]; then
        base64 /dev/urandom | head -c 1048576 >"$FILE_1_MB"
    fi

    if [ ! -e "$FILE_65_MB" ]; then
        base64 /dev/urandom | head -c 68157440 >"$FILE_65_MB"
    fi

    set -E
    set -o pipefail

    FILE_0_B_MD5SUM="$(get_md5sum "$FILE_0_B")"
    if [ $? -ne 0 ]; then
        echo "unable to get md5sum of $FILE_0_B"
        exit 1
    fi

    FILE_1_MB_MD5SUM="$(get_md5sum "$FILE_1_MB")"
    if [ $? -ne 0 ]; then
        echo "unable to get md5sum of $FILE_1_MB"
        exit 1
    fi

    FILE_65_MB_MD5SUM="$(get_md5sum "$FILE_65_MB")"
    if [ $? -ne 0 ]; then
        echo "unable to get md5sum of $FILE_65_MB"
        exit 1
    fi
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd alias set "${SERVER_ALIAS}" "$ENDPOINT" "$ACCESS_KEY" "$SECRET_KEY"
    assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd alias set "${SERVER_ALIAS_TLS}" "$ENDPOINT" "$ACCESS_KEY" "$SECRET_KEY"

    set +e
}

function main()
{
    ( run_test )
    rv=$?

    rm -fr "$MC_CONFIG_DIR" "$WATCH_OUT_FILE"
    if [ -z "$MINT_MODE" ]; then
        rm -fr "$WORK_DIR" "$DATA_DIR"
    fi

    exit "$rv"
}

__init__ "$@"
main "$@"
