#!/bin/bash


# Testing minioc cat
# 1. Pass a variable as stdin
# 2. Assert result returned by minioc cat is the same
# 3. Repeat the same for minioc pipe with no arguments
# 4. Take a random string to test for non existent files

dummy_string="asdasd"

cat_result=`minioc cat <<< $dummy_string`
if [ "$cat_result" == "$dummy_string" ]; then
    echo "minioc cat STDIN Test Passed"
else
    echo "minioc cat STDIN Test Failed"
fi

pipe_result=`minioc pipe <<< $dummy_string`
if [ "$pipe_result" == "$dummy_string" ]; then
    echo "minioc pipe STDIN Test Passed"
else
    echo "minioc pipe STDIN Test Failed"
fi

random_string="asd.jasd"

cat_json_result=`minioc cat --json $random_string`
if [[ "$cat_json_result" == *"\"status\":\"error\""* ]]
then
    echo "minioc cat Non-Existent File Test Passed"
else
    echo "minioc cat Non-Existent File Test Failed"
fi
