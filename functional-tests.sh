#!/bin/bash
#
# Copyright (c) 2015-2024 MinIO, Inc.
#
# This file is part of MinIO Object Storage stack
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.
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

if [ -n "$DEBUG" ]; then
	set -x
fi

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

# If you want to run the complete site cleaning test, set this variable to true
COMPLETE_RB_TEST=true

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

function get_md5sum() {
	filename="$1"
	out=$(md5sum "$filename" 2>/dev/null)
	rv=$?
	if [ "$rv" -eq 0 ]; then
		echo $(awk '{ print $1 }' <<<"$out")
	fi

	return "$rv"
}

function get_time() {
	date +%s%N
}

function get_duration() {
	start_time=$1
	end_time=$(get_time)

	echo $(((end_time - start_time) / 1000000))
}

function log_success() {
	if [ -n "$MINT_MODE" ]; then
		printf '{"name": "mc", "duration": "%d", "function": "%s", "status": "PASS"}\n' "$(get_duration "$1")" "$2"
	fi
}

function show() {
	if [ -z "$MINT_MODE" ]; then
		func_name="$1"
		echo "Running $func_name()"
	fi
}

function show_on_success() {
	rv="$1"
	shift

	if [ "$rv" -eq 0 ]; then
		echo "$@"
	fi

	return "$rv"
}

function show_on_failure() {
	rv="$1"
	shift

	if [ "$rv" -ne 0 ]; then
		echo "$@"
	fi

	return "$rv"
}

function assert() {
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

function mc_cmd() {
	cmd=("${MC_CMD[@]}" "$@")
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

function check_md5sum() {
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

function test_make_bucket() {
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

function test_rb() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	bucket1="mc-test-bucket-$RANDOM-1"
	bucket2="mc-test-bucket-$RANDOM-2"
	object_name="mc-test-object-$RANDOM"

	# Tets rb when the bucket is empty
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${bucket1}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rb "${SERVER_ALIAS}/${bucket1}"

	# Test rb with --force flag when the bucket is not empty
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${bucket1}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${bucket1}/${object_name}"
	assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd rb "${SERVER_ALIAS}/${bucket1}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rb --force "${SERVER_ALIAS}/${bucket1}"

	# Test rb with --force and --dangerous to remove a site content
	if [ "${COMPLETE_RB_TEST}" == "true" ]; then
		assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${bucket1}"
		assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${bucket2}"
		assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${bucket1}/${object_name}"
		assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${bucket2}/${object_name}"
		assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd rb --force "${SERVER_ALIAS}/"
		assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rb --force --dangerous "${SERVER_ALIAS}"
	fi

	log_success "$start_time" "${FUNCNAME[0]}"
}

function setup() {
	start_time=$(get_time)
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${BUCKET_NAME}"
}

function teardown() {
	start_time=$(get_time)
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rb --force "${SERVER_ALIAS}/${BUCKET_NAME}"
}

# Test mc ls on a S3 prefix where a lower similar prefix exists as well e.g. dir-foo/ and dir/
function test_list_dir() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/dir-foo/object1"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/dir/object2"
	diff -bB <(echo "object2") <("${MC_CMD[@]}" --json ls "${SERVER_ALIAS}/${BUCKET_NAME}/dir" | jq -r '.key') >/dev/null 2>&1
	assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unexpected listing dir"

	# Cleanup
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/dir-foo/${object_name}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/dir/${object_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_od_object() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd od if="${FILE_1_MB}" of="${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd od of="${FILE_1_MB}" if="${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_put_object() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_put_object_error() {
	show "${FUNCNAME[0]}"
	start_time=$(get_time)

	object_long_name=$(printf "mc-test-object-%01100d" 1)
	assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_long_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_put_object_multipart() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_65_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_put_object_0byte() {
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
function test_put_object_with_storage_class() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --storage-class REDUCED_REDUNDANCY "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

## Test mc cp command with storage-class flag set to incorrect value
function test_put_object_with_storage_class_error() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd cp --storage-class REDUCED "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

## Test mc cp command with valid metadata string
function test_put_object_with_metadata() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --attr key1=val1\;key2=val2 "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	diff -bB <(echo "val1") <("${MC_CMD[@]}" --json stat "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" | jq -r '.metadata."X-Amz-Meta-Key1"') >/dev/null 2>&1
	assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to put object with metadata"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	log_success "$start_time" "${FUNCNAME[0]}"

}

function test_get_object() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" "${object_name}.downloaded"
	assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${object_name}.downloaded"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_get_object_multipart() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_65_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" "${object_name}.downloaded"
	assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_65_MB_MD5SUM" "${object_name}.downloaded"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_presigned_post_policy_error() {
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

function test_presigned_put_object() {
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

function test_presigned_get_object() {
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

function test_cat_object() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	"${MC_CMD[@]}" cat "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" >"${object_name}.downloaded"
	assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to download object using 'mc cat'"
	assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${object_name}.downloaded"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_cat_stdin() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	bucket_name="mc-test-bucket-$RANDOM"
	mc_cmd mb "${SERVER_ALIAS}/${bucket_name}"
	echo "testcontent" | mc_cmd pipe "${SERVER_ALIAS}/${bucket_name}/${object_name}"
	"${MC_CMD[@]}" cat "${SERVER_ALIAS}/${bucket_name}/${object_name}" >stdout.output
	assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to redirect stdin to stdout using 'mc cat'"
	assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "42ed9fb3563d8e9c7bb522be443033f4" stdout.output
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm stdout.output

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_mirror_list_objects() {
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
function test_mirror_list_objects_storage_class() {
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

function test_watch_object() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	bucket_name="mc-test-bucket-$RANDOM"
	object_name="mc-test-object-$RANDOM"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${bucket_name}"

	# start a process to watch on bucket
	"${MC_CMD[@]}" --json watch "${SERVER_ALIAS}/${bucket_name}" >"$WATCH_OUT_FILE" &
	watch_cmd_pid=$!
	sleep 1

	(assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${bucket_name}/${object_name}")
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

	(assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${bucket_name}/${object_name}")
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

function test_config_host_add() {
	show "${FUNCNAME[0]}"
	start_time=$(get_time)

	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd alias set "${SERVER_ALIAS}1" "$ENDPOINT" "$ACCESS_KEY" "$SECRET_KEY"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd alias list "${SERVER_ALIAS}1"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_config_host_add_error() {
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

function test_put_object_with_sse() {
	show "${FUNCNAME[0]}"
	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"
	# put encrypted object; then delete with correct secret key
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --enc-c "${cli_flag}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_put_object_with_sse_error() {
	show "${FUNCNAME[0]}"
	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MD"
	# put object with invalid encryption key; should fail
	assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd cp --enc-c "${cli_flag}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_cat_object_with_sse() {
	show "${FUNCNAME[0]}"
	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"
	# put encrypted object; then cat object correct secret key
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --enc-c "${cli_flag}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cat --enc-c "${cli_flag}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_cat_object_with_sse_error() {
	show "${FUNCNAME[0]}"
	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"
	# put encrypted object; then cat object with no secret key
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --enc-c "${cli_flag}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd cat "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	log_success "$start_time" "${FUNCNAME[0]}"
}

# Test "mc cp -r" command of a directory with and without a leading slash
function test_copy_directory() {
	show "${FUNCNAME[0]}"

	random_dir="dir-$RANDOM-$RANDOM"
	tmpdir="$(mktemp -d)"

	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${random_dir}/object-name"
	assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to upload an object"

	# Copy a directory with a trailing slash
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp -r "${SERVER_ALIAS}/${BUCKET_NAME}/${random_dir}/" "${tmpdir}/"
	assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to copy a directory with a trailing slash"
	assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${tmpdir}/object-name"
	assert_success "$start_time" "${FUNCNAME[0]}" rm "${tmpdir}/object-name"

	# Copy a directory without a trailing slash
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp -r "${SERVER_ALIAS}/${BUCKET_NAME}/${random_dir}" "${tmpdir}/"
	assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to copy a directory without a trailing slash"
	assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${tmpdir}/${random_dir}/object-name"

	# Cleanup
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm -r --force "${SERVER_ALIAS}/${BUCKET_NAME}/${random_dir}/"
	assert_success "$start_time" "${FUNCNAME[0]}" rm -r "${tmpdir}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

# Test "mc cp -a" command to see if it preserves file system attributes
function test_copy_object_preserve_filesystem_attr() {
	show "${FUNCNAME[0]}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp -a "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	diff -bB <("${MC_CMD[@]}" --json stat "${FILE_1_MB}" | jq -r '.metadata."X-Amz-Meta-Mc-Attrs"') <("${MC_CMD[@]}" --json stat "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" | jq -r '.metadata."X-Amz-Meta-Mc-Attrs"') >/dev/null 2>&1 >/dev/null 2>&1
	assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to put object with file system attribute"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	log_success "$start_time" "${FUNCNAME[0]}"
}

# Test "mc mv" command
function test_mv_object() {
	show "${FUNCNAME[0]}"

	random_dir="dir-$RANDOM-$RANDOM"
	tmpdir="$(mktemp -d)"

	# Test mv command locally
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "${FILE_1_MB}" "${tmpdir}/file.tmp"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mv "${tmpdir}/file.tmp" "${tmpdir}/file"
	assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd stat "${tmpdir}/file.tmp"
	assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${tmpdir}/file"

	# Test mv command from filesystem to S3
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mv "${tmpdir}/file" "${SERVER_ALIAS}/${BUCKET_NAME}/${random_dir}/object-1"
	assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd stat "${tmpdir}/file"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd stat "${SERVER_ALIAS}/${BUCKET_NAME}/${random_dir}/object-1"

	# Test mv command from S3 to S3
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mv "${SERVER_ALIAS}/${BUCKET_NAME}/${random_dir}/object-1" "${SERVER_ALIAS}/${BUCKET_NAME}/${random_dir}/object-2"
	assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd stat "${SERVER_ALIAS}/${BUCKET_NAME}/${random_dir}/object-1"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd stat "${SERVER_ALIAS}/${BUCKET_NAME}/${random_dir}/object-2"

	# Test mv command from S3 to filesystem
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mv "${SERVER_ALIAS}/${BUCKET_NAME}/${random_dir}/object-2" "${tmpdir}/file"
	assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd stat "${SERVER_ALIAS}/${BUCKET_NAME}/${random_dir}/object-2"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd stat "${tmpdir}/file"

	# Cleanup
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm -r --force "${SERVER_ALIAS}/${BUCKET_NAME}/${random_dir}/"
	assert_success "$start_time" "${FUNCNAME[0]}" rm -r "${tmpdir}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_copy_object_with_sse_rewrite() {
	# test server side copy and remove operation - target is unencrypted while source is encrypted
	show "${FUNCNAME[0]}"
	start_time=$(get_time)
	prefix="prefix"
	object_name="mc-test-object-$RANDOM"

	cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"
	# create encrypted object on server
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --enc-c "${cli_flag}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}/${object_name}"
	# now do a server side copy and store it unencrypted.
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --enc-c "${cli_flag}" "${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}/${object_name}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	# cat the unencrypted destination object. should return data without any error
	"${MC_CMD[@]}" cat "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" >"${object_name}.downloaded"
	assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to download object using 'mc cat'"
	assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${object_name}.downloaded"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded"
	# mc rm on with multi-object delete, deletes encrypted object without encryption key.
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}/${object_name}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_copy_object_with_sse_dest() {
	# test server side copy and remove operation - target is encrypted with different key
	show "${FUNCNAME[0]}"
	start_time=$(get_time)
	prefix="prefix"
	object_name="mc-test-object-$RANDOM"

	cli_flag1="${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"
	cli_flag2="${SERVER_ALIAS}/${BUCKET_NAME}=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"

	# create encrypted object on server
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --enc-c "${cli_flag1}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}/${object_name}"
	# now do a server side copy and store it eith different encryption key.
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --enc-c "${cli_flag2}" \
		"${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}/${object_name}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	# cat the destination object with the new key. should return data without any error
	"${MC_CMD[@]}" cat --enc-c "${cli_flag2}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" >"${object_name}.downloaded"
	assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to download object using 'mc cat'"
	assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${object_name}.downloaded"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded"
	# mc rm on src object with first encryption key should pass
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}/${object_name}"
	# mc rm on encrypted destination object with second encryption key should pass
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_sse_key_rotation() {
	# test server side copy and remove operation - target is encrypted with different key
	show "${FUNCNAME[0]}"
	start_time=$(get_time)
	prefix="prefix"
	object_name="mc-test-object-$RANDOM"
	old_key="MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"
	new_key="MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"
	cli_flag1="${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}=${old_key}"
	cli_flag2="${SERVER_ALIAS_TLS}/${BUCKET_NAME}=${new_key}"

	# create encrypted object on server
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --enc-c "${cli_flag1}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${prefix}/${object_name}"
	# now do a server side copy on same object and do a key rotation
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --enc-c "${cli_flag2}" \
		"${SERVER_ALIAS_TLS}/${BUCKET_NAME}/${prefix}/${object_name}" "${SERVER_ALIAS_TLS}/${BUCKET_NAME}/${object_name}"
	# cat the object with the new key. should return data without any error
	"${MC_CMD[@]}" cat --enc-c "${cli_flag2}" "${SERVER_ALIAS_TLS}/${BUCKET_NAME}/${object_name}" >"${object_name}.downloaded"
	assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "unable to download object using 'mc cat'"
	assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${object_name}.downloaded"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded"
	# mc rm on encrypted object with succeed anyways, without encrypted keys.
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS_TLS}/${BUCKET_NAME}/${object_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_mirror_with_sse() {
	# test if mirror operation works with encrypted objects
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	bucket_name="mc-test-bucket-$RANDOM"
	cli_flag="${SERVER_ALIAS}/${bucket_name}=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"

	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mb "${SERVER_ALIAS}/${bucket_name}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd mirror --enc-c "${cli_flag}" "$DATA_DIR" "${SERVER_ALIAS}/${bucket_name}"
	diff -bB <(ls "$DATA_DIR") <("${MC_CMD[@]}" --json ls "${SERVER_ALIAS}/${bucket_name}/" | jq -r .key) >/dev/null 2>&1
	assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure $? "mirror and list differs"
	# Remove the test bucket with its contents
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rb --force "${SERVER_ALIAS}/${bucket_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_rm_object_with_sse() {
	show "${FUNCNAME[0]}"

	# test whether remove fails for encrypted object if secret key not provided.
	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"

	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --enc-c "${cli_flag}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	# rm will not fail even if the encryption keys are not provided, since mc rm uses multi-object delete.
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_get_object_with_sse() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"

	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --enc-c "${cli_flag}" "${FILE_1_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --enc-c "${cli_flag}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}" "${object_name}.downloaded"
	assert_success "$start_time" "${FUNCNAME[0]}" check_md5sum "$FILE_1_MB_MD5SUM" "${object_name}.downloaded"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${object_name}.downloaded" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_put_object_multipart_sse() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)
	object_name="mc-test-object-$RANDOM"
	cli_flag="${SERVER_ALIAS}/${BUCKET_NAME}=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"

	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp --enc-c "${cli_flag}" "${FILE_65_MB}" "${SERVER_ALIAS}/${BUCKET_NAME}/${object_name}"
	log_success "$start_time" "${FUNCNAME[0]}"
}

function test_admin_users() {
	show "${FUNCNAME[0]}"

	start_time=$(get_time)

	# create a user
	username=foo
	password=foobar12345
	test_alias="aliasx"

	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd admin user add "$SERVER_ALIAS" "$username" "$password"

	# check that user appears in the user list
	"${MC_CMD[@]}" --json admin user list "${SERVER_ALIAS}" | jq -r '.accessKey' | grep --quiet "^${username}$"
	rv=$?
	assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure ${rv} "user ${username} did NOT appear in the list of users returned by server"

	# setup temporary alias to make requests as the created user.
	scheme="https"
	if [ "$ENABLE_HTTPS" != "1" ]; then
		scheme="http"
	fi
	object1_name="mc-test-object-$RANDOM"
	object2_name="mc-test-object-$RANDOM"

	# Adding an alias for the $test_alias
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd alias set $test_alias "${scheme}://${SERVER_ENDPOINT}" ${username} ${password}

	# check that alias appears in the alias list
	"${MC_CMD[@]}" --json alias list | jq -r '.alias' | grep --quiet "^${test_alias}$"
	rv=$?
	assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure ${rv} "alias ${test_alias} did NOT appear in the list of aliases returned by server"

	# check that the user can write objects with readwrite policy
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd admin policy attach "$SERVER_ALIAS" readwrite --user="${username}"

	# verify that re-attaching an already attached policy to a user does not result in a failure.
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd admin policy attach "$SERVER_ALIAS" readwrite --user="${username}"

	# Validate that the correct policy has been added to the user
	"${MC_CMD[@]}" --json admin user list "${SERVER_ALIAS}" | jq -r '.policyName' | grep --quiet "^readwrite$"
	rv=$?
	assert_success "$start_time" "${FUNCNAME[0]}" show_on_failure ${rv} "user ${username} did NOT have the readwrite policy attached"

	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "$FILE_1_MB" "${test_alias}/${BUCKET_NAME}/${object1_name}"

	# check that the user cannot write objects with readonly policy
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd admin policy detach "$SERVER_ALIAS" readwrite --user="${username}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd admin policy attach "$SERVER_ALIAS" readonly --user="${username}"
	assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd cp "$FILE_1_MB" "${test_alias}/${BUCKET_NAME}/${object2_name}"

	# check that the user can read with readonly policy
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cat "${test_alias}/${BUCKET_NAME}/${object1_name}"

	# check that user can delete with readwrite policy
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd admin policy attach "$SERVER_ALIAS" readwrite --user="${username}"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd rm "${test_alias}/${BUCKET_NAME}/${object1_name}"

	# check that user cannot perform admin actions with readwrite policy
	assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd admin info $test_alias

	# create object1_name for subsequent tests.
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cp "$FILE_1_MB" "${test_alias}/${BUCKET_NAME}/${object1_name}"

	# check that user can be disabled
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd admin user disable "$SERVER_ALIAS" "$username"

	# check that disabled cannot perform any action
	assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd cat "${test_alias}/${BUCKET_NAME}/${object1_name}"

	# check that user can be enabled and can then perform an allowed action
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd admin user enable "$SERVER_ALIAS" "$username"
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd cat "${test_alias}/${BUCKET_NAME}/${object1_name}"

	# check that user can be removed, and then is no longer available
	assert_success "$start_time" "${FUNCNAME[0]}" mc_cmd admin user remove "$SERVER_ALIAS" "$username"
	assert_failure "$start_time" "${FUNCNAME[0]}" mc_cmd cat "${test_alias}/${BUCKET_NAME}/${object1_name}"

	log_success "$start_time" "${FUNCNAME[0]}"
}

function run_test() {
	test_make_bucket
	test_make_bucket_error
	test_rb

	setup
	test_list_dir
	test_put_object
	test_put_object_error
	test_put_object_0byte
	test_put_object_with_storage_class
	test_put_object_with_storage_class_error
	test_put_object_with_metadata
	test_put_object_multipart
	test_get_object
	test_get_object_multipart
	test_od_object
	test_mv_object
	test_presigned_post_policy_error
	test_presigned_put_object
	test_presigned_get_object
	test_cat_object
	test_cat_stdin
	test_copy_directory
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
	test_admin_users

	teardown
}

function __init__() {
	set -e
	if [ -n "$DEBUG" ]; then
		set -x
	fi

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
	MC_CMD=("${MC}" --config-dir "$MC_CONFIG_DIR" --quiet --no-color)

	if [ ! -e "$FILE_0_B" ]; then
		touch "$FILE_0_B"
	fi

	if [ ! -e "$FILE_1_MB" ]; then
		base64 -i /dev/urandom | head -c 1048576 >"$FILE_1_MB"
	fi

	if [ ! -e "$FILE_65_MB" ]; then
		base64 -i /dev/urandom | head -c 68157440 >"$FILE_65_MB"
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

function validate_dependencies() {
	jqVersion=$(jq --version)
	if [[ $jqVersion == *"jq"* ]]; then
		echo "Dependency validation complete"
	else
		echo "jq is missing, please install: 'sudo apt install jq'"
		exit 1
	fi
}

function main() {
	validate_dependencies

	(run_test)
	rv=$?

	rm -fr "$MC_CONFIG_DIR" "$WATCH_OUT_FILE"
	if [ -z "$MINT_MODE" ]; then
		rm -fr "$WORK_DIR" "$DATA_DIR"
	fi

	exit "$rv"
}

__init__ "$@"
main "$@"
