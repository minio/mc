#!/bin/bash


# Testing mc config  hosts
# 1. Get number of previous hosts
# 2. Add a host
# 3. Assert the increase in number of hosts by one.
# 4. Delete the new host


initial_count=`mc config host list | wc -l`
add_test_result=`mc config --json host add testdisk  https://storage.googleapis.com BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 S3v2`
final_count=`mc config host list | wc -l`
remove_test_result=`mc config host remove testdisk`

if [ "$add_test_result" == "Added ‘testdisk’ successfully." ]; then
    echo "mc config host add Test Passed";
else
    echo "mc config host add Test Failed";
fi

if [ $((initial_count + 1)) -ne $final_count ]; then
    echo "mc config host list Test Failed";
else
    echo "mc config host list Test Passed";
fi

if [ "$remove_test_result" == "Removed ‘testdisk’ successfully." ]; then
    echo "mc config host remove Test Passed";
else
    echo "mc config host remove Test Failed";
fi
