/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package minio

import (
	"fmt"
	"strings"

	"github.com/minio/minio-go/pkg/set"
)

// maximum supported access policy size.
const maxAccessPolicySize = 20 * 1024 * 1024 // 20KiB.

// Resource prefix for all aws resources.
const awsResourcePrefix = "arn:aws:s3:::"

// BucketPolicy - Bucket level policy.
type BucketPolicy string

// Different types of Policies currently supported for buckets.
const (
	BucketPolicyNone      BucketPolicy = "none"
	BucketPolicyReadOnly               = "readonly"
	BucketPolicyReadWrite              = "readwrite"
	BucketPolicyWriteOnly              = "writeonly"
)

// isValidBucketPolicy - Is provided policy value supported.
func (p BucketPolicy) isValidBucketPolicy() bool {
	switch p {
	case BucketPolicyNone, BucketPolicyReadOnly, BucketPolicyReadWrite, BucketPolicyWriteOnly:
		return true
	}
	return false
}

// User - canonical users list.
type User struct {
	AWS set.StringSet
}

// Statement - minio policy statement
type Statement struct {
	Sid        string
	Effect     string
	Principal  User                         `json:"Principal"`
	Actions    set.StringSet                `json:"Action"`
	Resources  set.StringSet                `json:"Resource"`
	Conditions map[string]map[string]string `json:"Condition,omitempty"`
}

// BucketAccessPolicy - minio policy collection
type BucketAccessPolicy struct {
	Version    string      // date in 0000-00-00 format
	Statements []Statement `json:"Statement"`
}

// Read write actions.
var (
	readWriteBucketActions = []string{
		"s3:GetBucketLocation",
		"s3:ListBucketMultipartUploads",
		// Add more bucket level read-write actions here.
	}
	readWriteObjectActions = []string{
		"s3:AbortMultipartUpload",
		"s3:DeleteObject",
		"s3:GetObject",
		"s3:ListMultipartUploadParts",
		"s3:PutObject",
		// Add more object level read-write actions here.
	}
)

// Write only actions.
var (
	writeOnlyBucketActions = []string{
		"s3:GetBucketLocation",
		"s3:ListBucketMultipartUploads",
		// Add more bucket level write actions here.
	}
	writeOnlyObjectActions = []string{
		"s3:AbortMultipartUpload",
		"s3:DeleteObject",
		"s3:ListMultipartUploadParts",
		"s3:PutObject",
		// Add more object level write actions here.
	}
)

// Read only actions.
var (
	readOnlyBucketActions = []string{
		"s3:GetBucketLocation",
		// Add more bucket level read actions here.
	}
	readOnlyObjectActions = []string{
		"s3:GetObject",
		// Add more object level read actions here.
	}
)

// Verifies if we have read/write policy set at bucketName, objectPrefix.
func isBucketPolicyReadWrite(statements []Statement, bucketName string, objectPrefix string) bool {
	rwBucketActions := set.CreateStringSet(readWriteBucketActions...)
	rwObjectActions := set.CreateStringSet(readWriteObjectActions...)
	bucketResource := awsResourcePrefix + bucketName
	objectResource := awsResourcePrefix + bucketName + "/" + objectPrefix
	var commonActions, readWrite bool
	for _, statement := range statements {
		if !statement.Principal.AWS.Contains("*") {
			continue
		}
		if statement.Resources.Contains(bucketResource) {
			if statement.Actions.Intersection(rwBucketActions).Equals(rwBucketActions) {
				commonActions = true
			}
		} else {
			resources := statement.Resources.FuncMatch(resourceMatch, objectResource)
			if !resources.IsEmpty() {
				if statement.Actions.Intersection(rwObjectActions).Equals(rwObjectActions) {
					readWrite = true
				}
			}
		}
	}
	return commonActions && readWrite
}

// Verifies if we have write only policy set at bucketName, objectPrefix.
func isBucketPolicyWriteOnly(statements []Statement, bucketName string, objectPrefix string) bool {
	woBucketActions := set.CreateStringSet(writeOnlyBucketActions...)
	woObjectActions := set.CreateStringSet(writeOnlyObjectActions...)
	bucketResource := awsResourcePrefix + bucketName
	objectResource := awsResourcePrefix + bucketName + "/" + objectPrefix
	var commonActions, writeOnly bool
	for _, statement := range statements {
		if !statement.Principal.AWS.Contains("*") {
			continue
		}

		if statement.Resources.Contains(bucketResource) {
			if statement.Actions.Intersection(woBucketActions).Equals(woBucketActions) {
				commonActions = true
			}
		} else {
			resources := statement.Resources.FuncMatch(resourceMatch, objectResource)
			if !resources.IsEmpty() {
				if statement.Actions.Intersection(woObjectActions).Equals(woObjectActions) {
					writeOnly = true
				}
			}
		}
	}
	return commonActions && writeOnly
}

// Verifies if we have read only policy set at bucketName, objectPrefix.
func isBucketPolicyReadOnly(statements []Statement, bucketName string, objectPrefix string) bool {
	roBucketActions := set.CreateStringSet(readOnlyBucketActions...)
	roObjectActions := set.CreateStringSet(readOnlyObjectActions...)
	bucketResource := awsResourcePrefix + bucketName
	objectResource := awsResourcePrefix + bucketName + "/" + objectPrefix
	var commonActions, readOnly bool
	for _, statement := range statements {
		if !statement.Principal.AWS.Contains("*") {
			continue
		}

		if statement.Resources.Contains(bucketResource) {
			if statement.Actions.Intersection(roBucketActions).Equals(roBucketActions) {
				commonActions = true
			}
		} else {
			resources := statement.Resources.FuncMatch(resourceMatch, objectResource)
			if !resources.IsEmpty() {
				if statement.Actions.Intersection(roObjectActions).Equals(roObjectActions) {
					readOnly = true
				}
			}
		}
	}
	return commonActions && readOnly
}

// removeReadBucketActions - removes readWriteBucket actions if found.
func removeReadBucketActions(statements []Statement, bucketName string) []Statement {
	rwBucketActions := set.CreateStringSet(readWriteBucketActions...)
	bucketResource := awsResourcePrefix + bucketName
	var newStatements []Statement
	var bucketActionsRemoved bool
	for _, statement := range statements {
		if statement.Resources.Contains(bucketResource) && !bucketActionsRemoved {
			statement.Actions = statement.Actions.Difference(rwBucketActions)
			bucketActionsRemoved = true
		}
		if !statement.Actions.IsEmpty() {
			newStatements = append(newStatements, statement)
		}
	}
	return newStatements
}

// removeListBucketActions - removes "s3:ListBucket" action if found.
func removeListBucketAction(statements []Statement, bucketName string) []Statement {
	bucketResource := awsResourcePrefix + bucketName
	var newStatements []Statement
	var listBucketActionsRemoved bool
	for _, statement := range statements {
		if statement.Resources.Contains(bucketResource) && !listBucketActionsRemoved {
			if statement.Actions.Contains("s3:ListBucket") {
				delete(statement.Conditions, "StringEquals")
				statement.Actions.Remove("s3:ListBucket")
				listBucketActionsRemoved = true
			}
		}
		if !statement.Actions.IsEmpty() {
			newStatements = append(newStatements, statement)
		}
	}
	return newStatements
}

// removeWriteObjectActions - removes writeOnlyObject actions if found.
func removeWriteObjectActions(statements []Statement, bucketName string, objectPrefix string) []Statement {
	woObjectActions := set.CreateStringSet(writeOnlyObjectActions...)
	objectResource := awsResourcePrefix + bucketName + "/" + objectPrefix + "*"
	var newStatements []Statement
	for _, statement := range statements {
		if statement.Resources.Contains(objectResource) {
			statement.Actions = statement.Actions.Difference(woObjectActions)
		}
		if !statement.Actions.IsEmpty() {
			newStatements = append(newStatements, statement)
		}
	}
	return newStatements
}

// removeReadObjectActions - removes "s3:GetObject" actions if found.
func removeReadObjectActions(statements []Statement, bucketName string, objectPrefix string) []Statement {
	objectResource := awsResourcePrefix + bucketName + "/" + objectPrefix + "*"
	var newStatements []Statement
	for _, statement := range statements {
		if statement.Resources.Contains(objectResource) {
			statement.Actions.Remove("s3:GetObject")
		}
		if !statement.Actions.IsEmpty() {
			newStatements = append(newStatements, statement)
		}
	}
	return newStatements
}

// removeReadWriteObjectActions - removes readWriteObject actions if found.
func removeReadWriteObjectActions(statements []Statement, bucketName string, objectPrefix string) []Statement {
	rwObjectActions := set.CreateStringSet(readWriteObjectActions...)
	objectResource := awsResourcePrefix + bucketName + "/" + objectPrefix + "*"
	var newStatements []Statement
	for _, statement := range statements {
		if statement.Resources.Contains(objectResource) {
			statement.Actions = statement.Actions.Difference(rwObjectActions)
		}
		if !statement.Actions.IsEmpty() {
			newStatements = append(newStatements, statement)
		}
	}
	return newStatements
}

// Removes read write bucket policy if found.
func removeBucketPolicyStatementReadWrite(statements []Statement, bucketName string, objectPrefix string) []Statement {
	newStatements := removeReadBucketActions(statements, bucketName)
	newStatements = removeListBucketAction(newStatements, bucketName)
	newStatements = removeReadWriteObjectActions(newStatements, bucketName, objectPrefix)
	return newStatements
}

// Removes write only bucket policy if found.
func removeBucketPolicyStatementWriteOnly(statements []Statement, bucketName string, objectPrefix string) []Statement {
	newStatements := removeReadBucketActions(statements, bucketName)
	newStatements = removeWriteObjectActions(newStatements, bucketName, objectPrefix)
	return newStatements
}

// Removes read only bucket policy if found.
func removeBucketPolicyStatementReadOnly(statements []Statement, bucketName string, objectPrefix string) []Statement {
	newStatements := removeReadBucketActions(statements, bucketName)
	newStatements = removeListBucketAction(newStatements, bucketName)
	newStatements = removeReadObjectActions(newStatements, bucketName, objectPrefix)
	return newStatements
}

// Remove bucket policies based on the type.
func removeBucketPolicyStatement(statements []Statement, bucketName string, objectPrefix string) []Statement {
	// Verify that a policy is defined on the object prefix, otherwise do not remove the policy
	if isPolicyDefinedForObjectPrefix(statements, bucketName, objectPrefix) {
		// Verify type of policy to be removed.
		if isBucketPolicyReadWrite(statements, bucketName, objectPrefix) {
			statements = removeBucketPolicyStatementReadWrite(statements, bucketName, objectPrefix)
		} else if isBucketPolicyWriteOnly(statements, bucketName, objectPrefix) {
			statements = removeBucketPolicyStatementWriteOnly(statements, bucketName, objectPrefix)
		} else if isBucketPolicyReadOnly(statements, bucketName, objectPrefix) {
			statements = removeBucketPolicyStatementReadOnly(statements, bucketName, objectPrefix)
		}
	}
	return statements
}

// Checks if an access policiy is defined for the given object prefix
func isPolicyDefinedForObjectPrefix(statements []Statement, bucketName string, objectPrefix string) bool {
	objectResource := awsResourcePrefix + bucketName + "/" + objectPrefix + "*"
	for _, statement := range statements {
		if statement.Resources.Contains(objectResource) {
			return true
		}
	}
	return false
}

// Identifies the policy type from policy Statements.
func identifyPolicyType(policy BucketAccessPolicy, bucketName, objectPrefix string) (bucketPolicy BucketPolicy) {
	if policy.Statements == nil {
		return BucketPolicyNone
	}
	if isBucketPolicyReadWrite(policy.Statements, bucketName, objectPrefix) {
		return BucketPolicyReadWrite
	} else if isBucketPolicyWriteOnly(policy.Statements, bucketName, objectPrefix) {
		return BucketPolicyWriteOnly
	} else if isBucketPolicyReadOnly(policy.Statements, bucketName, objectPrefix) {
		return BucketPolicyReadOnly
	}
	return BucketPolicyNone
}

// Generate policy statements for various bucket policies.
// refer to http://docs.aws.amazon.com/AmazonS3/latest/dev/access-policy-language-overview.html
// for more details about statement fields.
func generatePolicyStatement(bucketPolicy BucketPolicy, bucketName, objectPrefix string) ([]Statement, error) {
	if !bucketPolicy.isValidBucketPolicy() {
		return []Statement{}, ErrInvalidArgument(fmt.Sprintf("Invalid bucket policy provided. %s", bucketPolicy))
	}
	var statements []Statement
	if bucketPolicy == BucketPolicyNone {
		return []Statement{}, nil
	} else if bucketPolicy == BucketPolicyReadWrite {
		// Get read-write policy.
		statements = setReadWriteStatement(bucketName, objectPrefix)
	} else if bucketPolicy == BucketPolicyReadOnly {
		// Get read only policy.
		statements = setReadOnlyStatement(bucketName, objectPrefix)
	} else if bucketPolicy == BucketPolicyWriteOnly {
		// Return Write only policy.
		statements = setWriteOnlyStatement(bucketName, objectPrefix)
	}
	return statements, nil
}

// Obtain statements for read-write BucketPolicy.
func setReadWriteStatement(bucketName, objectPrefix string) []Statement {
	bucketResourceStatement := Statement{}
	bucketResourceStatement.Effect = "Allow"
	bucketResourceStatement.Principal.AWS = set.CreateStringSet("*")
	bucketResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName)
	bucketResourceStatement.Actions = set.CreateStringSet(readWriteBucketActions...)

	bucketListResourceStatement := Statement{}
	bucketListResourceStatement.Effect = "Allow"
	bucketListResourceStatement.Principal.AWS = set.CreateStringSet("*")
	bucketListResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName)
	bucketListResourceStatement.Actions = set.CreateStringSet("s3:ListBucket")
	// Object prefix is present, make sure to set the conditions for s3:ListBucket.
	if objectPrefix != "" {
		bucketListResourceStatement.Conditions = map[string]map[string]string{
			"StringEquals": {
				"s3:prefix": objectPrefix,
			},
		}
	}
	objectResourceStatement := Statement{}
	objectResourceStatement.Effect = "Allow"
	objectResourceStatement.Principal.AWS = set.CreateStringSet("*")
	objectResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName + "/" + objectPrefix + "*")
	objectResourceStatement.Actions = set.CreateStringSet(readWriteObjectActions...)
	// Save the read write policy.
	statements := []Statement{}
	statements = append(statements, bucketResourceStatement, bucketListResourceStatement, objectResourceStatement)
	return statements
}

// Obtain statements for read only BucketPolicy.
func setReadOnlyStatement(bucketName, objectPrefix string) []Statement {
	bucketResourceStatement := Statement{}
	bucketResourceStatement.Effect = "Allow"
	bucketResourceStatement.Principal.AWS = set.CreateStringSet("*")
	bucketResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName)
	bucketResourceStatement.Actions = set.CreateStringSet(readOnlyBucketActions...)

	bucketListResourceStatement := Statement{}
	bucketListResourceStatement.Effect = "Allow"
	bucketListResourceStatement.Principal.AWS = set.CreateStringSet("*")
	bucketListResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName)
	bucketListResourceStatement.Actions = set.CreateStringSet("s3:ListBucket")
	// Object prefix is present, make sure to set the conditions for s3:ListBucket.
	if objectPrefix != "" {
		bucketListResourceStatement.Conditions = map[string]map[string]string{
			"StringEquals": {
				"s3:prefix": objectPrefix,
			},
		}
	}
	objectResourceStatement := Statement{}
	objectResourceStatement.Effect = "Allow"
	objectResourceStatement.Principal.AWS = set.CreateStringSet("*")
	objectResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName + "/" + objectPrefix + "*")
	objectResourceStatement.Actions = set.CreateStringSet(readOnlyObjectActions...)

	statements := []Statement{}

	// Save the read only policy.
	statements = append(statements, bucketResourceStatement, bucketListResourceStatement, objectResourceStatement)
	return statements
}

// Obtain statements for write only BucketPolicy.
func setWriteOnlyStatement(bucketName, objectPrefix string) []Statement {
	bucketResourceStatement := Statement{}
	objectResourceStatement := Statement{}
	statements := []Statement{}
	// Write only policy.
	bucketResourceStatement.Effect = "Allow"
	bucketResourceStatement.Principal.AWS = set.CreateStringSet("*")
	bucketResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName)
	bucketResourceStatement.Actions = set.CreateStringSet(writeOnlyBucketActions...)
	objectResourceStatement.Effect = "Allow"
	objectResourceStatement.Principal.AWS = set.CreateStringSet("*")
	objectResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName + "/" + objectPrefix + "*")
	objectResourceStatement.Actions = set.CreateStringSet(writeOnlyObjectActions...)
	// Save the write only policy.
	statements = append(statements, bucketResourceStatement, objectResourceStatement)
	return statements
}

// Match function matches wild cards in 'pattern' for resource.
func resourceMatch(pattern, resource string) bool {
	if pattern == "" {
		return resource == pattern
	}
	if pattern == "*" {
		return true
	}
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return resource == pattern
	}
	tGlob := strings.HasSuffix(pattern, "*")
	end := len(parts) - 1
	if !strings.HasPrefix(resource, parts[0]) {
		return false
	}
	for i := 1; i < end; i++ {
		if !strings.Contains(resource, parts[i]) {
			return false
		}
		idx := strings.Index(resource, parts[i]) + len(parts[i])
		resource = resource[idx:]
	}
	return tGlob || strings.HasSuffix(resource, parts[end])
}
