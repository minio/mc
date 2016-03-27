#!/bin/bash


# Testing mc cat
# 1. Pass a variable as stdin
# 2. Assert result returned by mc cat is the same
# 3. Repeat the same for mc pipe with no arguments
# 4. Take a random string to test for non existant files

dummy_string="asdasd"

cat_result=`mc cat <<< $dummy_string`
if [ "$cat_result" == "$dummy_string" ]; then
    echo "mc cat STDIN Test Passed"
else
    echo "mc cat STDIN Test Failed"
fi

pipe_result=`mc pipe <<< $dummy_string`
if [ "$pipe_result" == "$dummy_string" ]; then
    echo "mc pipe STDIN Test Passed"
else
    echo "mc pipe STDIN Test Failed"
fi

random_string="asd.jasd"

cat_json_result=`mc cat --json $random_string`
if [[ "$cat_json_result" == *"\"status\":\"error\""* ]]
then
    echo "mc cat Non-Existent File Test Passed"
else
    echo "mc cat Non-Existent File Test Failed"
fi
