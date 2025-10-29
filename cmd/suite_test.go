// Copyright (c) 2015-2024 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/mc/pkg/disk"
)

// RUN: go test -v ./... -run Test_FullSuite
func Test_FullSuite(t *testing.T) {
	if os.Getenv("MC_TEST_RUN_FULL_SUITE") != "true" {
		return
	}

	defer func() {
		r := recover()
		if r != nil {
			log.Println(r, string(debug.Stack()))
		}

		postRunCleanup(t)
	}()

	preflightCheck(t)
	// initializeTestSuite builds the mc client and creates local files which are used for testing
	initializeTestSuite(t)

	// Tests within this function depend on one another
	testsThatDependOnOneAnother(t)

	// Alias tests
	AddALIASWithError(t)

	// Basic admin user tests
	AdminUserFunctionalTest(t)

	// Share upload/download
	ShareURLUploadTest(t)
	ShareURLDownloadTest(t)

	// TODO .. for some reason the connection is randomly
	// reset when running curl.
	// ShareURLUploadErrorTests(t)

	// Bucket Error Tests
	CreateBucketUsingInvalidSymbols(t)
	RemoveBucketWithNameTooLong(t)
	RemoveBucketThatDoesNotExist(t)

	// MC_TEST_ENABLE_HTTPS=true
	// needs to be set in order to run these tests
	if protocol == "https://" {
		PutObjectWithSSECHexKey(t)
		GetObjectWithSSEC(t)

		PutObjectWithSSEC(t)
		PutObjectWithSSECPartialPrefixMatch(t)
		PutObjectWithSSECMultipart(t)
		PutObjectWithSSECInvalidKeys(t)
		GetObjectWithSSEC(t)
		GetObjectWithSSECWithoutKey(t)
		CatObjectWithSSEC(t)
		CatObjectWithSSECWithoutKey(t)
		CopyObjectWithSSECToNewBucketWithNewKey(t)
		MirrorTempDirectoryUsingSSEC(t)
		RemoveObjectWithSSEC(t)
	} else {
		PutObjectErrorWithSSECOverHTTP(t)
	}

	// MC_TEST_KMS_KEY=[KEY_NAME]
	// needs to be set in order to run these tests
	if sseKMSKeyName != "" {
		VerifyKMSKey(t)
		PutObjectWithSSEKMS(t)
		PutObjectWithSSEKMSPartialPrefixMatch(t)
		PutObjectWithSSEKMSMultipart(t)
		PutObjectWithSSEKMSInvalidKeys(t)
		GetObjectWithSSEKMS(t)
		CatObjectWithSSEKMS(t)
		CopyObjectWithSSEKMSToNewBucket(t)
		MirrorTempDirectoryUsingSSEKMS(t)
		RemoveObjectWithSSEKMS(t)

		// Error tests
		CopyObjectWithSSEKMSWithOverLappingKeys(t)
	}

	// MC_TEST_ENABLE_SSE_S3=true
	// needs to be set to in order to run these tests.
	if sseS3Enabled {
		PutObjectWithSSES3(t)
		PutObjectWithSSES3PartialPrefixMatch(t)
		PutObjectWithSSES3Multipart(t)
		GetObjectWithSSES3(t)
		CatObjectWithSSES3(t)
		CopyObjectWithSSES3ToNewBucket(t)
		MirrorTempDirectoryUsingSSES3(t)
	}

	if protocol == "https://" && sseKMSKeyName != "" {
		CopyObjectWithSSEKMSToNewBucketWithSSEC(t)
	}

	// (DEPRECATED CLI PARAMETERS)
	if includeDeprecatedMethods {
		fmt.Println("No deprecated methods implemented")
	}
}

func testsThatDependOnOneAnother(t *testing.T) {
	CreateFileBundle()
	// uploadAllFiles uploads all files in FileMap to MainTestBucket
	uploadAllFiles(t)
	// LSObjects saves the output of LS inside *testFile in FileMap
	LSObjects(t)
	// StatObjecsts saves the output of Stat inside *testFile in FileMap
	StatObjects(t)
	// ValidateFileMetaDataPostUpload validates the output of LS and Stat
	ValidateFileMetaData(t)

	// DU tests
	DUBucket(t)

	// Std in/out .. pipe/cat
	CatObjectToStdIn(t)
	CatObjectFromStdin(t)

	// Preserve attributes
	PutObjectPreserveAttributes(t)

	// Mirror
	MirrorTempDirectoryStorageClassReducedRedundancy(t)
	MirrorTempDirectory(t)
	MirrorMinio2MinioWithTagsCopy(t)

	// General object tests
	FindObjects(t)
	FindObjectsUsingName(t)
	FindObjectsUsingNameAndFilteringForTxtType(t)
	FindObjectsLargerThan64Mebibytes(t)
	FindObjectsSmallerThan64Mebibytes(t)
	FindObjectsOlderThan1d(t)
	FindObjectsNewerThen1d(t)
	GetObjectsAndCompareMD5(t)
}

type TestUser struct {
	Username string
	Password string
}

var (
	oneMBSlice               [1048576]byte // 1x Mebibyte
	defaultAlias             = "mintest"
	fileMap                  = make(map[string]*testFile)
	randomLargeString        = "lksdjfljsdklfjklsdjfklksjdf;lsjdk;fjks;djflsdlfkjskldjfklkljsdfljsldkfjklsjdfkljsdklfjklsdjflksjdlfjsdjflsjdflsldfjlsjdflksjdflkjslkdjflksfdj"
	jsonFlag                 = "--json"
	insecureFlag             = "--insecure"
	jsonOutput               = true
	printRawOut              = false
	skipBuild                = false
	mcCmd                    = ".././mc"
	preCmdParameters         = make([]string, 0)
	buildPath                = "../."
	metaPrefix               = "X-Amz-Meta-"
	includeDeprecatedMethods = false

	serverEndpoint = "127.0.0.1:9000"
	acessKey       = "minioadmin"
	secretKey      = "minioadmin"
	protocol       = "http://"
	skipInsecure   = true
	tempDir        = ""
	mainTestBucket string
	sseTestBucket  string
	bucketList     = make([]string, 0)
	userList       = make(map[string]TestUser, 0)

	// ENCRYPTION
	sseHexKey                = "8fe4d820587c427d5cc207d75cb76f3c6874808174b04050fa209206bfd08ebb"
	sseBaseEncodedKey        = "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"
	invalidSSEBaseEncodedKey = "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5"
	sseBaseEncodedKey2       = "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5YWE"
	sseKMSKeyName            = ""
	sseInvalidKmsKeyName     = ""
	sseS3Enabled             = false

	curlPath      = "/usb/bin/curl"
	HTTPClient    *http.Client
	failIndicator = "!! FAIL !! _______________________ !! FAIL !! _______________________ !! FAIL !!"
)

func openFileAndGetMd5Sum(path string) (md5s string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	fb, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	md5s = GetMD5Sum(fb)
	return
}

func GetMBSizeInBytes(MB int) int64 {
	return int64(MB * len(oneMBSlice))
}

func initializeTestSuite(t *testing.T) {
	shouldSkipBuild := os.Getenv("MC_TEST_SKIP_BUILD")
	skipBuild, _ = strconv.ParseBool(shouldSkipBuild)
	fmt.Println("SKIP BUILD:", skipBuild)
	if !skipBuild {
		err := BuildCLI()
		if err != nil {
			os.Exit(1)
		}
	}
	envBuildPath := os.Getenv("MC_TEST_BUILD_PATH")
	if envBuildPath != "" {
		buildPath = envBuildPath
	}

	envALIAS := os.Getenv("MC_TEST_ALIAS")
	if envALIAS != "" {
		defaultAlias = envALIAS
	}

	envSecretKey := os.Getenv("MC_TEST_SECRET_KEY")
	if envSecretKey != "" {
		secretKey = envSecretKey
	}

	envAccessKey := os.Getenv("MC_TEST_ACCESS_KEY")
	if envAccessKey != "" {
		acessKey = envAccessKey
	}

	envServerEndpoint := os.Getenv("MC_TEST_SERVER_ENDPOINT")
	if envServerEndpoint != "" {
		serverEndpoint = envServerEndpoint
	}

	envIncludeDeprecated := os.Getenv("MC_TEST_INCLUDE_DEPRECATED")
	includeDeprecatedMethods, _ = strconv.ParseBool(envIncludeDeprecated)

	envKmsKey := os.Getenv("MC_TEST_KMS_KEY")
	if envKmsKey != "" {
		sseKMSKeyName = envKmsKey
	}

	envSSES3Enabled := os.Getenv("MC_TEST_ENABLE_SSE_S3")
	if envSSES3Enabled != "" {
		sseS3Enabled, _ = strconv.ParseBool(envSSES3Enabled)
	}

	envSkipInsecure := os.Getenv("MC_TEST_SKIP_INSECURE")
	if envSkipInsecure != "" {
		skipInsecure, _ = strconv.ParseBool(envSkipInsecure)
	}

	envEnableHTTP := os.Getenv("MC_TEST_ENABLE_HTTPS")
	EnableHTTPS, _ := strconv.ParseBool(envEnableHTTP)
	if EnableHTTPS {
		protocol = "https://"
	}

	envCMD := os.Getenv("MC_TEST_BINARY_PATH")
	if envCMD != "" {
		mcCmd = envCMD
	}

	var err error
	tempDir, err = os.MkdirTemp("", "test-")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	for i := range len(oneMBSlice) {
		oneMBSlice[i] = byte(rand.Intn(250))
	}

	for i := range 10 {
		tmpNameMap["aaa"+strconv.Itoa(i)] = false
	}
	for i := range 10 {
		tmpNameMap["bbb"+strconv.Itoa(i)] = false
	}
	for i := range 10 {
		tmpNameMap["ccc"+strconv.Itoa(i)] = false
	}
	for i := range 10 {
		tmpNameMap["ddd"+strconv.Itoa(i)] = false
	}

	HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: skipInsecure},
		},
	}

	if jsonOutput {
		preCmdParameters = append(preCmdParameters, jsonFlag)
	}

	if skipInsecure {
		preCmdParameters = append(preCmdParameters, insecureFlag)
	}

	CreateTestUsers()

	_, err = RunMC(
		"alias",
		"set",
		defaultAlias,
		protocol+serverEndpoint,
		acessKey,
		secretKey,
	)
	fatalIfError(err, t)

	out, err := RunMC("--version")
	fatalIfError(err, t)
	fmt.Println(out)

	preRunCleanup()

	mainTestBucket = CreateBucket(t)
	sseTestBucket = CreateBucket(t)
}

func preflightCheck(t *testing.T) {
	out, err := exec.Command("which", "curl").Output()
	fatalIfError(err, t)
	if len(out) == 0 {
		fatalMsgOnly("No curl found, output from 'which curl': "+string(out), t)
	}
	curlPath = string(out)
}

func CreateTestUsers() {
	userList["user1"] = TestUser{
		Username: "user1",
		Password: "user1-password",
	}
	userList["user2"] = TestUser{
		Username: "user2",
		Password: "user2-password",
	}
	userList["user3"] = TestUser{
		Username: "user3",
		Password: "user3-password",
	}
}

func CreateFileBundle() {
	createFile(newTestFile{
		tag:          "0M",
		prefix:       "",
		extension:    ".jpg",
		storageClass: "",
		sizeInMBS:    0,
		tags:         map[string]string{"name": "0M"},
		// uploadShouldFail:   false,
		addToGlobalFileMap: true,
	})
	createFile(newTestFile{
		tag:          "1M",
		prefix:       "",
		extension:    ".txt",
		storageClass: "REDUCED_REDUNDANCY",
		sizeInMBS:    1,
		metaData:     map[string]string{"name": "1M"},
		tags:         map[string]string{"tag1": "1M-tag"},
		// uploadShouldFail:   false,
		addToGlobalFileMap: true,
	})
	createFile(newTestFile{
		tag:          "2M",
		prefix:       "LVL1",
		extension:    ".jpg",
		storageClass: "REDUCED_REDUNDANCY",
		sizeInMBS:    2,
		metaData:     map[string]string{"name": "2M"},
		// uploadShouldFail:   false,
		addToGlobalFileMap: true,
	})
	createFile(newTestFile{
		tag:          "3M",
		prefix:       "LVL1/LVL2",
		extension:    ".png",
		storageClass: "",
		sizeInMBS:    3,
		metaData:     map[string]string{"name": "3M"},
		// uploadShouldFail:   false,
		addToGlobalFileMap: true,
	})
	createFile(newTestFile{
		tag:          "65M",
		prefix:       "LVL1/LVL2/LVL3",
		extension:    ".exe",
		storageClass: "",
		sizeInMBS:    65,
		metaData:     map[string]string{"name": "65M", "tag1": "value1"},
		// uploadShouldFail:   false,
		addToGlobalFileMap: true,
	})
}

var tmpNameMap = make(map[string]bool)

func GetRandomName() string {
	for i := range tmpNameMap {
		if tmpNameMap[i] == false {
			tmpNameMap[i] = true
			return i
		}
	}
	return uuid.NewString()
}

func CreateBucket(t *testing.T) (bucketPath string) {
	bucketName := "test-" + GetRandomName()
	bucketPath = defaultAlias + "/" + bucketName
	out, err := RunMC("mb", bucketPath)
	if err != nil {
		t.Fatalf("Unable to create bucket (%s) err: %s", bucketPath, out)
		return
	}
	bucketList = append(bucketList, bucketPath)
	out, err = RunMC("stat", defaultAlias+"/"+bucketName)
	if err != nil {
		t.Fatalf("Unable to ls stat (%s) err: %s", defaultAlias+"/"+bucketName, out)
		return
	}
	if !strings.Contains(out, bucketName) {
		t.Fatalf("stat output does not contain bucket name (%s)", bucketName)
	}
	return
}

func AddALIASWithError(t *testing.T) {
	out, err := RunMC(
		"alias",
		"set",
		defaultAlias,
		protocol+serverEndpoint,
		acessKey,
		"random-invalid-secret-that-will-not-work",
	)
	fatalIfNoErrorWMsg(err, out, t)
}

func AdminUserFunctionalTest(t *testing.T) {
	user1Bucket := CreateBucket(t)

	user1File := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "user1",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"admin",
		"user",
		"add",
		defaultAlias,
		userList["user1"].Username,
		userList["user1"].Password,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"admin",
		"user",
		"list",
		defaultAlias,
	)
	fatalIfErrorWMsg(err, out, t)
	userOutput, err := parseUserMessageListOutput(out)
	fatalIfErrorWMsg(err, out, t)

	user1found := false
	for i := range userOutput {
		if userOutput[i].AccessKey == userList["user1"].Username {
			user1found = true
		}
	}

	if !user1found {
		fatalMsgOnly(fmt.Sprintf("did not find user %s when running admin user list --json", userList["user1"].Username), t)
	}

	out, err = RunMC(
		"admin",
		"policy",
		"attach",
		defaultAlias,
		"readwrite",
		"--user="+userList["user1"].Username,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"alias",
		"set",
		userList["user1"].Username,
		protocol+serverEndpoint,
		userList["user1"].Username,
		userList["user1"].Password,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"cp",
		user1File.diskFile.Name(),
		user1Bucket+"/"+user1File.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)
}

func ShareURLUploadErrorTests(t *testing.T) {
	shareURLErrorBucket := CreateBucket(t)

	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "presigned-error",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"share",
		"upload",
		shareURLErrorBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	shareMsg, err := parseShareMessageFromJSONOutput(out)
	fatalIfErrorWMsg(err, out, t)

	finalURL := strings.ReplaceAll(shareMsg.ShareURL, "<FILE>", file.diskFile.Name())
	splitCommand := strings.Split(finalURL, " ")

	if skipInsecure {
		splitCommand = append(splitCommand, "--insecure")
	}

	bucketOnly := strings.ReplaceAll(shareURLErrorBucket, defaultAlias+"/", "")

	// Modify base url bucket path
	newCmd := make([]string, len(splitCommand))
	copy(newCmd, splitCommand)
	newCmd[1] = strings.ReplaceAll(newCmd[1], bucketOnly, "fake-bucket-name")
	out, _ = RunCommand(newCmd[0], newCmd[1:]...)
	curlFatalIfNoErrorTag(out, t)

	// Modify -F key=X
	newCmd = make([]string, len(splitCommand))
	copy(newCmd, splitCommand)
	for i := range newCmd {
		if strings.HasPrefix(newCmd[i], "key=") {
			newCmd[i] = "key=fake-object-name"
			break
		}
	}
	out, _ = RunCommand(newCmd[0], newCmd[1:]...)
	curlFatalIfNoErrorTag(out, t)
}

func ShareURLUploadTest(t *testing.T) {
	ShareURLTestBucket := CreateBucket(t)

	file := createFile(newTestFile{
		addToGlobalFileMap: true,
		tag:                "presigned-upload",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"share",
		"upload",
		ShareURLTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	shareMsg, err := parseShareMessageFromJSONOutput(out)
	fatalIfErrorWMsg(err, out, t)

	finalURL := strings.ReplaceAll(shareMsg.ShareURL, "<FILE>", file.diskFile.Name())
	splitCommand := strings.Split(finalURL, " ")

	if skipInsecure {
		splitCommand = append(splitCommand, "--insecure")
	}

	_, err = exec.Command(splitCommand[0], splitCommand[1:]...).CombinedOutput()
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"stat",
		ShareURLTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	statMsg, err := parseStatSingleObjectJSONOutput(out)
	fatalIfError(err, t)

	if statMsg.ETag != file.md5Sum {
		fatalMsgOnly(fmt.Sprintf("expecting md5sum (%s) but got md5sum (%s)", file.md5Sum, file.md5Sum), t)
	}
}

func ShareURLDownloadTest(t *testing.T) {
	ShareURLTestBucket := CreateBucket(t)
	file := createFile(newTestFile{
		addToGlobalFileMap: true,
		tag:                "presigned-download",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		file.diskFile.Name(),
		ShareURLTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"share",
		"download",
		ShareURLTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	shareMsg, err := parseShareMessageFromJSONOutput(out)
	fatalIfErrorWMsg(err, out, t)

	resp, err := HTTPClient.Get(shareMsg.ShareURL)
	fatalIfError(err, t)

	downloadedFile, err := io.ReadAll(resp.Body)
	fatalIfError(err, t)

	md5sum := GetMD5Sum(downloadedFile)
	if md5sum != file.md5Sum {
		fatalMsgOnly(
			fmt.Sprintf("expecting md5sum (%s) but got md5sum (%s)", file.md5Sum, md5sum),
			t,
		)
	}
}

func PutObjectPreserveAttributes(t *testing.T) {
	AttrTestBucket := CreateBucket(t)
	file := fileMap["1M"]
	out, err := RunMC(
		"cp",
		"-a",
		file.diskFile.Name(),
		AttrTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"stat",
		AttrTestBucket+"/"+file.fileNameWithPrefix,
	)
	fatalIfError(err, t)

	stats, err := parseStatSingleObjectJSONOutput(out)
	fatalIfError(err, t)

	attr, err := disk.GetFileSystemAttrs(file.diskFile.Name())
	fatalIfError(err, t)
	if attr != stats.Metadata["X-Amz-Meta-Mc-Attrs"] {
		fatalMsgOnly(fmt.Sprintf("expecting file attributes (%s) but got file attributes (%s)", attr, stats.Metadata["X-Amz-Meta-Mc-Attrs"]), t)
	}
}

func MirrorTempDirectoryStorageClassReducedRedundancy(t *testing.T) {
	MirrorBucket := CreateBucket(t)
	out, err := RunMC(
		"mirror",
		"--storage-class", "REDUCED_REDUNDANCY",
		tempDir,
		MirrorBucket,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC("ls", "-r", MirrorBucket)
	fatalIfError(err, t)

	fileList, err := parseLSJSONOutput(out)
	fatalIfError(err, t)

	for i, f := range fileMap {
		fileFound := false

		for _, o := range fileList {
			if o.Key == f.fileNameWithoutPath {
				fileMap[i].MinioLS = o
				fileFound = true
			}
		}

		if !fileFound {
			t.Fatalf("File was not uploaded: %s", f.fileNameWithPrefix)
		}
	}
}

func MirrorTempDirectory(t *testing.T) {
	MirrorBucket := CreateBucket(t)

	out, err := RunMC(
		"mirror",
		tempDir,
		MirrorBucket,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC("ls", "-r", MirrorBucket)
	fatalIfError(err, t)

	fileList, err := parseLSJSONOutput(out)
	fatalIfError(err, t)

	for i, f := range fileMap {
		fileFound := false

		for _, o := range fileList {
			if o.Key == f.fileNameWithoutPath {
				fileMap[i].MinioLS = o
				fileFound = true
			}
		}

		if !fileFound {
			t.Fatalf("File was not uploaded: %s", f.fileNameWithPrefix)
		}
	}
}

type tagListResult struct {
	Tagset    map[string]string `json:"tagset"`
	Status    string            `json:"status"`
	URL       string            `json:"url"`
	VersionID string            `json:"versionID"`
}

func MirrorMinio2MinioWithTagsCopy(t *testing.T) {
	TargetBucket := CreateBucket(t)
	out, err := RunMC(
		"mirror",
		mainTestBucket,
		TargetBucket,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC("tag", "list", TargetBucket, "-r")
	fatalIfErrorWMsg(err, out, t)
	var targetTagsList []tagListResult
	outStrs := bufio.NewScanner(strings.NewReader(out))
	for outStrs.Scan() {
		outStr := outStrs.Text()
		if outStr == "" {
			continue
		}
		tagList := new(tagListResult)
		if err := json.Unmarshal([]byte(outStr), tagList); err != nil {
			fatalIfErrorWMsg(err, out, t)
		}
		targetTagsList = append(targetTagsList, *tagList)
	}

	for _, f := range fileMap {
		fileFound := false

		for _, o := range targetTagsList {
			if strings.Contains(o.URL, f.fileNameWithPrefix) {
				fileFound = true
				if !reflect.DeepEqual(f.tags, o.Tagset) {
					fatalMsgOnly(fmt.Sprintf("expecting tags (%s) but got tags (%s)", f.tags, o.Tagset), t)
				}
			}
		}

		if !fileFound {
			t.Fatalf("File was not mirrored: %s", f.fileNameWithPrefix)
		}
	}
}

func CatObjectFromStdin(t *testing.T) {
	objectName := "pipe-test-object"
	CatEchoBucket := CreateBucket(t)

	file := fileMap["1M"]

	cmdCAT := exec.Command(
		"cat",
		file.diskFile.Name(),
	)

	p := []string{
		"pipe",
		CatEchoBucket + "/" + objectName,
	}
	if skipInsecure {
		p = append(p, "--insecure")
	}

	cmdMC := exec.Command(mcCmd, p...)

	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()

	cmdCAT.Stdout = w
	cmdMC.Stdin = r

	err := cmdMC.Start()
	fatalIfError(err, t)
	err = cmdCAT.Start()
	fatalIfError(err, t)

	err = cmdCAT.Wait()
	fatalIfError(err, t)
	w.Close()
	err = cmdMC.Wait()
	fatalIfError(err, t)
	r.Close()

	outB, err := RunMC(
		"cat",
		CatEchoBucket+"/"+objectName,
	)
	fatalIfErrorWMsg(err, outB, t)

	md5SumCat := GetMD5Sum([]byte(outB))
	if file.md5Sum != md5SumCat {
		fatalMsgOnly(
			fmt.Sprintf("expecting md5sum (%s) but got md5sum (%s)", file.md5Sum, md5SumCat),
			t,
		)
	}
}

func CatObjectToStdIn(t *testing.T) {
	file := fileMap["1M"]
	out, err := RunMC(
		"cat",
		mainTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)
	md5Sum := GetMD5Sum([]byte(out))
	if md5Sum != file.md5Sum {
		fatalMsgOnly(
			fmt.Sprintf("expecting md5sum (%s) but got md5sum (%s)", file.md5Sum, md5Sum),
			t,
		)
	}
}

func VerifyKMSKey(t *testing.T) {
	out, err := RunMC(
		"admin",
		"kms",
		"key",
		"list",
		defaultAlias,
	)
	fatalIfError(err, t)
	keyMsg := new(kmsKeysMsg)
	err = json.Unmarshal([]byte(out), keyMsg)
	fatalIfError(err, t)
	sseInvalidKmsKeyName = uuid.NewString()
	found := false
	invalidKeyFound := false
	for _, v := range keyMsg.Keys {
		if v == sseKMSKeyName {
			found = true
			break
		}
		if v == sseInvalidKmsKeyName {
			invalidKeyFound = true
		}
	}
	if !found {
		fatalMsgOnly(fmt.Sprintf("expected to find kms key %s but got these keys: %v", sseKMSKeyName, keyMsg.Keys), t)
	}
	if invalidKeyFound {
		fatalMsgOnly("tried to create invalid uuid kms key but for some reason it overlapped with an already existing key", t)
	}
}

func PutObjectWithSSEKMSPartialPrefixMatch(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encput-kms-prefix-test",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-kms",
		sseTestBucket+"/"+file.fileNameWithoutPath+"="+sseKMSKeyName,
		file.diskFile.Name(),
		sseTestBucket,
	)
	fatalIfErrorWMsg(err, out, t)
}

func PutObjectWithSSEKMS(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encput-kms",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-kms",
		sseTestBucket+"="+sseKMSKeyName,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)
}

func PutObjectWithSSEKMSMultipart(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encmultiput-kms",
		sizeInMBS:          68,
	})

	out, err := RunMC(
		"cp",
		"--enc-kms",
		sseTestBucket+"="+sseKMSKeyName,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)
}

func PutObjectWithSSEKMSInvalidKeys(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encerror-kms",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-kms="+sseTestBucket+"="+sseInvalidKmsKeyName,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfNoErrorWMsg(err, out, t)
}

func GetObjectWithSSES3(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encget-s3",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-s3="+sseTestBucket,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"cp",
		sseTestBucket+"/"+file.fileNameWithoutPath,
		file.diskFile.Name()+".download",
	)
	fatalIfErrorWMsg(err, out, t)

	md5s, err := openFileAndGetMd5Sum(file.diskFile.Name() + ".download")
	fatalIfError(err, t)
	if md5s != file.md5Sum {
		fatalMsgOnly(fmt.Sprintf("expecting md5sum (%s) but got sum (%s)", file.md5Sum, md5s), t)
	}
}

func CatObjectWithSSES3(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "enccat-s3",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-s3="+sseTestBucket,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"cat",
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)
	catMD5Sum := GetMD5Sum([]byte(out))

	if catMD5Sum != file.md5Sum {
		fatalMsgOnly(fmt.Sprintf(
			"expected md5sum %s but we got %s",
			file.md5Sum,
			catMD5Sum,
		), t)
	}

	if int64(len(out)) != file.diskStat.Size() {
		fatalMsgOnly(fmt.Sprintf(
			"file size is %d but we got %d",
			file.diskStat.Size(),
			len(out),
		), t)
	}

	fatalIfErrorWMsg(
		err,
		"cat length: "+strconv.Itoa(len(out))+" -- file length:"+strconv.Itoa(int(file.diskStat.Size())),
		t,
	)
}

func CopyObjectWithSSES3ToNewBucket(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encbucketcopy-s3",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-s3="+sseTestBucket,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	TargetSSEBucket := CreateBucket(t)

	out, err = RunMC(
		"cp",
		"--enc-s3="+TargetSSEBucket,
		sseTestBucket+"/"+file.fileNameWithoutPath,
		TargetSSEBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"cp",
		TargetSSEBucket+"/"+file.fileNameWithoutPath,
		file.diskFile.Name()+".download",
	)
	fatalIfErrorWMsg(err, out, t)

	md5s, err := openFileAndGetMd5Sum(file.diskFile.Name() + ".download")
	fatalIfError(err, t)
	if md5s != file.md5Sum {
		fatalMsgOnly(fmt.Sprintf("expecting md5sum (%s) but got sum (%s)", file.md5Sum, md5s), t)
	}
}

func MirrorTempDirectoryUsingSSES3(t *testing.T) {
	MirrorBucket := CreateBucket(t)

	subDir := "encmirror-s3"

	f1 := createFile(newTestFile{
		addToGlobalFileMap: false,
		subDir:             subDir,
		tag:                "encmirror1-s3",
		sizeInMBS:          1,
	})

	f2 := createFile(newTestFile{
		addToGlobalFileMap: false,
		subDir:             subDir,
		tag:                "encmirror2-s3",
		sizeInMBS:          2,
	})

	f3 := createFile(newTestFile{
		addToGlobalFileMap: false,
		subDir:             subDir,
		tag:                "encmirror3-s3",
		sizeInMBS:          4,
	})

	files := append([]*testFile{}, f1, f2, f3)

	out, err := RunMC(
		"mirror",
		"--enc-s3="+MirrorBucket,
		tempDir+string(os.PathSeparator)+subDir,
		MirrorBucket,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC("ls", "-r", MirrorBucket)
	fatalIfError(err, t)

	fileList, err := parseLSJSONOutput(out)
	fatalIfError(err, t)

	for i, f := range files {
		fileFound := false

		for _, o := range fileList {
			if o.Key == f.fileNameWithoutPath {
				files[i].MinioLS = o
				fileFound = true
			}
		}

		if !fileFound {
			fatalMsgOnly(fmt.Sprintf(
				"File was not uploaded: %s",
				f.fileNameWithPrefix,
			), t)
		}

		out, err := RunMC("stat", MirrorBucket+"/"+files[i].MinioLS.Key)
		fatalIfError(err, t)
		stat, err := parseStatSingleObjectJSONOutput(out)
		fatalIfError(err, t)
		files[i].MinioStat = stat

		foundKmsTag := false
		for ii := range stat.Metadata {
			if ii == amzObjectSSE {
				foundKmsTag = true
				break
			}
		}

		if !foundKmsTag {
			fmt.Println(stat)
			fatalMsgOnly(amzObjectSSEKMSKeyID+" not found for object "+files[i].MinioLS.Key, t)
		}

	}
}

func PutObjectWithSSES3PartialPrefixMatch(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encput-s3-prefix-test",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-s3="+sseTestBucket+"/"+file.fileNameWithoutPath,
		file.diskFile.Name(),
		sseTestBucket,
	)
	fatalIfErrorWMsg(err, out, t)
}

func PutObjectWithSSES3(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encput-s3",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-s3="+sseTestBucket,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)
}

func PutObjectWithSSES3Multipart(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encmultiput-s3",
		sizeInMBS:          68,
	})

	out, err := RunMC(
		"cp",
		"--enc-s3="+sseTestBucket,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)
}

func GetObjectWithSSEKMS(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encget-kms",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-kms="+sseTestBucket+"="+sseKMSKeyName,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"cp",
		sseTestBucket+"/"+file.fileNameWithoutPath,
		file.diskFile.Name()+".download",
	)
	fatalIfErrorWMsg(err, out, t)

	md5s, err := openFileAndGetMd5Sum(file.diskFile.Name() + ".download")
	fatalIfError(err, t)
	if md5s != file.md5Sum {
		fatalMsgOnly(fmt.Sprintf("expecting md5sum (%s) but got sum (%s)", file.md5Sum, md5s), t)
	}
}

func PutObjectWithSSECMultipart(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encmultiput",
		sizeInMBS:          68,
	})

	out, err := RunMC(
		"cp",
		"--enc-c="+sseTestBucket+"="+sseBaseEncodedKey,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)
}

func CatObjectWithSSEKMS(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "enccat-kms",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-kms="+sseTestBucket+"="+sseKMSKeyName,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"cat",
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)
	catMD5Sum := GetMD5Sum([]byte(out))

	if catMD5Sum != file.md5Sum {
		fatalMsgOnly(fmt.Sprintf(
			"expected md5sum %s but we got %s",
			file.md5Sum,
			catMD5Sum,
		), t)
	}

	if int64(len(out)) != file.diskStat.Size() {
		fatalMsgOnly(fmt.Sprintf(
			"file size is %d but we got %d",
			file.diskStat.Size(),
			len(out),
		), t)
	}

	fatalIfErrorWMsg(
		err,
		"cat length: "+strconv.Itoa(len(out))+" -- file length:"+strconv.Itoa(int(file.diskStat.Size())),
		t,
	)
}

func PutObjectWithSSECPartialPrefixMatch(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encput-prefix-test",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-c="+sseTestBucket+"/"+file.fileNameWithoutPath+"="+sseBaseEncodedKey,
		file.diskFile.Name(),
		sseTestBucket,
	)
	fatalIfErrorWMsg(err, out, t)
}

func PutObjectWithSSECHexKey(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encputhex",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-c="+sseTestBucket+"="+sseHexKey,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)
}

func PutObjectWithSSEC(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encput",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-c="+sseTestBucket+"="+sseBaseEncodedKey,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)
}

func PutObjectErrorWithSSECOverHTTP(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encput-http",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-c="+sseTestBucket+"="+sseBaseEncodedKey,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfNoErrorWMsg(err, out, t)
}

func PutObjectWithSSECInvalidKeys(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encerror-dep",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-c="+sseTestBucket+"="+invalidSSEBaseEncodedKey,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfNoErrorWMsg(err, out, t)
}

func GetObjectWithSSECHexKey(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encgethex",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-c="+sseTestBucket+"="+sseHexKey,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"cp",
		"--enc-c="+sseTestBucket+"="+sseHexKey,
		sseTestBucket+"/"+file.fileNameWithoutPath,
		file.diskFile.Name()+".download",
	)
	fatalIfErrorWMsg(err, out, t)

	md5s, err := openFileAndGetMd5Sum(file.diskFile.Name() + ".download")
	fatalIfError(err, t)
	if md5s != file.md5Sum {
		fatalMsgOnly(fmt.Sprintf("expecting md5sum (%s) but got sum (%s)", file.md5Sum, md5s), t)
	}
}

func GetObjectWithSSEC(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encget",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-c="+sseTestBucket+"="+sseBaseEncodedKey,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"cp",
		"--enc-c="+sseTestBucket+"="+sseBaseEncodedKey,
		sseTestBucket+"/"+file.fileNameWithoutPath,
		file.diskFile.Name()+".download",
	)
	fatalIfErrorWMsg(err, out, t)

	md5s, err := openFileAndGetMd5Sum(file.diskFile.Name() + ".download")
	fatalIfError(err, t)
	if md5s != file.md5Sum {
		fatalMsgOnly(fmt.Sprintf("expecting md5sum (%s) but got sum (%s)", file.md5Sum, md5s), t)
	}
}

func GetObjectWithSSECWithoutKey(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encerror",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-c="+sseTestBucket+"="+sseBaseEncodedKey,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"cp",
		sseTestBucket+"/"+file.fileNameWithoutPath,
		file.diskFile.Name()+"-get",
	)
	fatalIfNoErrorWMsg(err, out, t)
}

func CatObjectWithSSEC(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "enccat",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-c="+sseTestBucket+"="+sseBaseEncodedKey,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"cat",
		"--enc-c="+sseTestBucket+"="+sseBaseEncodedKey,
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)
	catMD5Sum := GetMD5Sum([]byte(out))

	if catMD5Sum != file.md5Sum {
		fatalMsgOnly(fmt.Sprintf(
			"expected md5sum %s but we got %s",
			file.md5Sum,
			catMD5Sum,
		), t)
	}

	if int64(len(out)) != file.diskStat.Size() {
		fatalMsgOnly(fmt.Sprintf(
			"file size is %d but we got %d",
			file.diskStat.Size(),
			len(out),
		), t)
	}

	fatalIfErrorWMsg(
		err,
		"cat length: "+strconv.Itoa(len(out))+" -- file length:"+strconv.Itoa(int(file.diskStat.Size())),
		t,
	)
}

func CopyObjectWithSSEKMSWithOverLappingKeys(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encbucketcopy-kms",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-kms="+sseTestBucket+"="+sseKMSKeyName,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	TargetSSEBucket := CreateBucket(t)

	out, err = RunMC(
		"cp",
		"--enc-kms="+TargetSSEBucket+"="+sseKMSKeyName,
		"--enc-kms="+TargetSSEBucket+"="+sseKMSKeyName,
		sseTestBucket+"/"+file.fileNameWithoutPath,
		TargetSSEBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfNoErrorWMsg(err, out, t)
}

func CopyObjectWithSSEKMSToNewBucket(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encbucketcopy-kms",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-kms="+sseTestBucket+"="+sseKMSKeyName,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	TargetSSEBucket := CreateBucket(t)

	out, err = RunMC(
		"cp",
		"--enc-kms="+TargetSSEBucket+"="+sseKMSKeyName,
		"--enc-kms="+sseTestBucket+"="+sseKMSKeyName,
		sseTestBucket+"/"+file.fileNameWithoutPath,
		TargetSSEBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"cp",
		"--enc-kms="+TargetSSEBucket+"="+sseKMSKeyName,
		TargetSSEBucket+"/"+file.fileNameWithoutPath,
		file.diskFile.Name()+".download",
	)
	fatalIfErrorWMsg(err, out, t)

	md5s, err := openFileAndGetMd5Sum(file.diskFile.Name() + ".download")
	fatalIfError(err, t)
	if md5s != file.md5Sum {
		fatalMsgOnly(fmt.Sprintf("expecting md5sum (%s) but got sum (%s)", file.md5Sum, md5s), t)
	}
}

func CopyObjectWithSSEKMSToNewBucketWithSSEC(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encbucketcopy-kms-c",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-kms="+sseTestBucket+"="+sseKMSKeyName,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	TargetSSEBucket := CreateBucket(t)

	out, err = RunMC(
		"cp",
		"--enc-c="+TargetSSEBucket+"="+sseBaseEncodedKey,
		sseTestBucket+"/"+file.fileNameWithoutPath,
		TargetSSEBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"cp",
		"--enc-c="+TargetSSEBucket+"="+sseBaseEncodedKey,
		TargetSSEBucket+"/"+file.fileNameWithoutPath,
		file.diskFile.Name()+".download",
	)
	fatalIfErrorWMsg(err, out, t)

	md5s, err := openFileAndGetMd5Sum(file.diskFile.Name() + ".download")
	fatalIfError(err, t)
	if md5s != file.md5Sum {
		fatalMsgOnly(fmt.Sprintf("expecting md5sum (%s) but got sum (%s)", file.md5Sum, md5s), t)
	}
}

func MirrorTempDirectoryUsingSSEKMS(t *testing.T) {
	MirrorBucket := CreateBucket(t)

	subDir := "encmirror-kms"

	f1 := createFile(newTestFile{
		addToGlobalFileMap: false,
		subDir:             subDir,
		tag:                "encmirror1-kms",
		sizeInMBS:          1,
	})

	f2 := createFile(newTestFile{
		addToGlobalFileMap: false,
		subDir:             subDir,
		tag:                "encmirror2-kms",
		sizeInMBS:          2,
	})

	f3 := createFile(newTestFile{
		addToGlobalFileMap: false,
		subDir:             subDir,
		tag:                "encmirror3-kms",
		sizeInMBS:          4,
	})

	files := append([]*testFile{}, f1, f2, f3)

	out, err := RunMC(
		"mirror",
		"--enc-kms="+MirrorBucket+"="+sseKMSKeyName,
		tempDir+string(os.PathSeparator)+subDir,
		MirrorBucket,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC("ls", "-r", MirrorBucket)
	fatalIfError(err, t)

	fileList, err := parseLSJSONOutput(out)
	fatalIfError(err, t)

	for i, f := range files {
		fileFound := false

		for _, o := range fileList {
			if o.Key == f.fileNameWithoutPath {
				files[i].MinioLS = o
				fileFound = true
			}
		}

		if !fileFound {
			fatalMsgOnly(fmt.Sprintf(
				"File was not uploaded: %s",
				f.fileNameWithPrefix,
			), t)
		}

		out, err := RunMC("stat", MirrorBucket+"/"+files[i].MinioLS.Key)
		fatalIfError(err, t)
		stat, err := parseStatSingleObjectJSONOutput(out)
		fatalIfError(err, t)
		files[i].MinioStat = stat

		foundKmsTag := false
		for ii, v := range stat.Metadata {
			if ii == amzObjectSSEKMSKeyID {
				foundKmsTag = true
				if !strings.HasSuffix(v, sseKMSKeyName) {
					fatalMsgOnly("invalid KMS key for object "+files[i].MinioLS.Key, t)
					break
				}
			}
		}

		if !foundKmsTag {
			fatalMsgOnly(amzObjectSSEKMSKeyID+" not found for object "+files[i].MinioLS.Key, t)
		}

	}
}

func RemoveObjectWithSSEKMS(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encrm-kms",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-kms="+sseTestBucket+"="+sseKMSKeyName,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"rm",
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"stat",
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfNoErrorWMsg(err, out, t)
}

func CatObjectWithSSECWithoutKey(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encerror",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-c="+sseTestBucket+"="+sseBaseEncodedKey,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"cat",
		sseTestBucket+"/"+file.fileNameWithoutPath,
		file.diskFile.Name()+"-cat",
	)
	fatalIfNoErrorWMsg(err, out, t)
}

func RemoveObjectWithSSEC(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encrm",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-c="+sseTestBucket+"="+sseBaseEncodedKey,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"rm",
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"stat",
		"--enc-c="+sseTestBucket+"="+sseBaseEncodedKey,
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfNoErrorWMsg(err, out, t)
}

func MirrorTempDirectoryUsingSSEC(t *testing.T) {
	MirrorBucket := CreateBucket(t)

	subDir := "encmirror"

	f1 := createFile(newTestFile{
		addToGlobalFileMap: false,
		subDir:             subDir,
		tag:                "encmirror1",
		sizeInMBS:          1,
	})

	f2 := createFile(newTestFile{
		addToGlobalFileMap: false,
		subDir:             subDir,
		tag:                "encmirror2",
		sizeInMBS:          2,
	})

	f3 := createFile(newTestFile{
		addToGlobalFileMap: false,
		subDir:             subDir,
		tag:                "encmirror3",
		sizeInMBS:          4,
	})

	files := append([]*testFile{}, f1, f2, f3)

	out, err := RunMC(
		"mirror",
		"--enc-c="+MirrorBucket+"="+sseBaseEncodedKey,
		tempDir+string(os.PathSeparator)+subDir,
		MirrorBucket,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC("ls", "-r", MirrorBucket)
	fatalIfError(err, t)

	fileList, err := parseLSJSONOutput(out)
	fatalIfError(err, t)

	for i, f := range files {
		fileFound := false

		for _, o := range fileList {
			if o.Key == f.fileNameWithoutPath {
				files[i].MinioLS = o
				fileFound = true
			}
		}

		if !fileFound {
			fatalMsgOnly(fmt.Sprintf(
				"File was not uploaded: %s",
				f.fileNameWithPrefix,
			), t)
		}

		out, err := RunMC(
			"stat",
			"--enc-c="+MirrorBucket+"="+sseBaseEncodedKey,
			MirrorBucket+"/"+files[i].MinioLS.Key,
		)
		fatalIfError(err, t)
		_, err = parseStatSingleObjectJSONOutput(out)
		fatalIfError(err, t)

	}
}

func CopyObjectWithSSECToNewBucketWithNewKey(t *testing.T) {
	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "encbucketcopy",
		sizeInMBS:          1,
	})

	out, err := RunMC(
		"cp",
		"--enc-c="+sseTestBucket+"="+sseBaseEncodedKey,
		file.diskFile.Name(),
		sseTestBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	TargetSSEBucket := CreateBucket(t)

	out, err = RunMC(
		"cp",
		"--enc-c="+TargetSSEBucket+"="+sseBaseEncodedKey2,
		"--enc-c="+sseTestBucket+"="+sseBaseEncodedKey,
		sseTestBucket+"/"+file.fileNameWithoutPath,
		TargetSSEBucket+"/"+file.fileNameWithoutPath,
	)
	fatalIfErrorWMsg(err, out, t)

	out, err = RunMC(
		"cp",
		"--enc-c="+TargetSSEBucket+"="+sseBaseEncodedKey2,
		TargetSSEBucket+"/"+file.fileNameWithoutPath,
		file.diskFile.Name()+".download",
	)
	fatalIfErrorWMsg(err, out, t)

	md5s, err := openFileAndGetMd5Sum(file.diskFile.Name() + ".download")
	fatalIfError(err, t)
	if md5s != file.md5Sum {
		fatalMsgOnly(fmt.Sprintf("expecting md5sum (%s) but got sum (%s)", file.md5Sum, md5s), t)
	}
}

func uploadAllFiles(t *testing.T) {
	for _, v := range fileMap {
		parameters := make([]string, 0)
		parameters = append(parameters, "cp")

		if v.storageClass != "" {
			parameters = append(parameters, "--storage-class", v.storageClass)
		}

		if len(v.metaData) > 0 {
			parameters = append(parameters, "--attr")
			meta := ""
			for i, v := range v.metaData {
				meta += i + "=" + v + ";"
			}
			meta = strings.TrimSuffix(meta, ";")
			parameters = append(parameters, meta)
		}
		if len(v.tags) > 0 {
			parameters = append(parameters, "--tags")
			tags := ""
			for i, v := range v.tags {
				tags += i + "=" + v + ";"
			}
			tags = strings.TrimSuffix(tags, ";")
			parameters = append(parameters, tags)
		}

		parameters = append(parameters, v.diskFile.Name())

		if v.prefix != "" {
			parameters = append(
				parameters,
				mainTestBucket+"/"+v.fileNameWithPrefix,
			)
		} else {
			parameters = append(
				parameters,
				mainTestBucket+"/"+v.fileNameWithoutPath,
			)
		}

		_, err := RunMC(parameters...)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func OD(t *testing.T) {
	LocalBucketPath := CreateBucket(t)

	file := fileMap["65M"]
	out, err := RunMC(
		"od",
		"if="+file.diskFile.Name(),
		"of="+LocalBucketPath+"/od/"+file.fileNameWithoutPath,
		"parts=10",
	)

	fatalIfError(err, t)
	odMsg, err := parseSingleODMessageJSONOutput(out)
	fatalIfError(err, t)

	if odMsg.TotalSize != file.diskStat.Size() {
		t.Fatalf(
			"Expected (%d) bytes to be uploaded but only uploaded (%d) bytes",
			odMsg.TotalSize,
			file.diskStat.Size(),
		)
	}

	if odMsg.Parts != 10 {
		t.Fatalf(
			"Expected upload parts to be (10) but they were (%d)",
			odMsg.Parts,
		)
	}

	if odMsg.Type != "FStoS3" {
		t.Fatalf(
			"Expected type to be (FStoS3) but got (%s)",
			odMsg.Type,
		)
	}

	if odMsg.PartSize != uint64(file.diskStat.Size())/10 {
		t.Fatalf(
			"Expected part size to be (%d) but got (%d)",
			file.diskStat.Size()/10,
			odMsg.PartSize,
		)
	}

	out, err = RunMC(
		"od",
		"of="+file.diskFile.Name(),
		"if="+LocalBucketPath+"/od/"+file.fileNameWithoutPath,
		"parts=10",
	)

	fatalIfError(err, t)
	fmt.Println(out)
	odMsg, err = parseSingleODMessageJSONOutput(out)
	fatalIfError(err, t)

	if odMsg.TotalSize != file.diskStat.Size() {
		t.Fatalf(
			"Expected (%d) bytes to be uploaded but only uploaded (%d) bytes",
			odMsg.TotalSize,
			file.diskStat.Size(),
		)
	}

	if odMsg.Parts != 10 {
		t.Fatalf(
			"Expected upload parts to be (10) but they were (%d)",
			odMsg.Parts,
		)
	}

	if odMsg.Type != "S3toFS" {
		t.Fatalf(
			"Expected type to be (FStoS3) but got (%s)",
			odMsg.Type,
		)
	}

	if odMsg.PartSize != uint64(file.diskStat.Size())/10 {
		t.Fatalf(
			"Expected part size to be (%d) but got (%d)",
			file.diskStat.Size()/10,
			odMsg.PartSize,
		)
	}
}

func MvFromDiskToMinio(t *testing.T) {
	LocalBucketPath := CreateBucket(t)

	file := createFile(newTestFile{
		addToGlobalFileMap: false,
		tag:                "10Move",
		prefix:             "",
		extension:          ".txt",
		storageClass:       "",
		sizeInMBS:          1,
		metaData:           map[string]string{"name": "10Move"},
		tags:               map[string]string{"tag1": "10Move-tag"},
	})

	out, err := RunMC(
		"mv",
		file.diskFile.Name(),
		LocalBucketPath+"/"+file.fileNameWithoutPath,
	)

	fatalIfError(err, t)
	splitReturn := bytes.Split([]byte(out), []byte{10})

	mvMSG, err := parseSingleCPMessageJSONOutput(string(splitReturn[0]))
	fatalIfError(err, t)

	if mvMSG.TotalCount != 1 {
		t.Fatalf("Expected count to be 1 but got (%d)", mvMSG.TotalCount)
	}

	if mvMSG.Size != file.diskStat.Size() {
		t.Fatalf(
			"Expected size to be (%d) but got (%d)",
			file.diskStat.Size(),
			mvMSG.Size,
		)
	}

	if mvMSG.Status != "success" {
		t.Fatalf(
			"Expected status to be (success) but got (%s)",
			mvMSG.Status,
		)
	}

	statMSG, err := parseSingleAccountStatJSONOutput(string(splitReturn[1]))
	fatalIfError(err, t)

	if statMSG.Transferred != file.diskStat.Size() {
		t.Fatalf(
			"Expected transfeered to be (%d) but got (%d)",
			file.diskStat.Size(),
			statMSG.Transferred,
		)
	}

	if statMSG.Total != file.diskStat.Size() {
		t.Fatalf(
			"Expected total to be (%d) but got (%d)",
			file.diskStat.Size(),
			statMSG.Total,
		)
	}

	if statMSG.Status != "success" {
		t.Fatalf(
			"Expected status to be (success) but got (%s)",
			statMSG.Status,
		)
	}
}

func DUBucket(t *testing.T) {
	var totalFileSize int64
	for _, v := range fileMap {
		totalFileSize += v.MinioStat.Size
	}

	out, err := RunMC("du", mainTestBucket)
	fatalIfError(err, t)

	duList, err := parseDUJSONOutput(out)
	fatalIfError(err, t)
	if len(duList) != 1 {
		fatalMsgOnly("Expected 1 result to be returned", t)
	}
	if duList[0].Size != totalFileSize {
		fatalMsgOnly(
			fmt.Sprintf("total size to be %d but got %d", totalFileSize, duList[0].Size),
			t,
		)
	}
}

func LSObjects(t *testing.T) {
	out, err := RunMC("ls", "-r", mainTestBucket)
	fatalIfError(err, t)

	fileList, err := parseLSJSONOutput(out)
	fatalIfError(err, t)

	for i, f := range fileMap {
		fileFound := false

		for _, o := range fileList {
			if o.Key == f.fileNameWithPrefix {
				fileMap[i].MinioLS = o
				fileFound = true
			}
		}

		if !fileFound {
			t.Fatalf("File was not uploaded: %s", f.fileNameWithPrefix)
		}
	}
}

func StatObjects(t *testing.T) {
	for i, v := range fileMap {

		out, err := RunMC(
			"stat",
			mainTestBucket+"/"+v.fileNameWithPrefix,
		)
		fatalIfError(err, t)

		fileMap[i].MinioStat, err = parseStatSingleObjectJSONOutput(out)
		fatalIfError(err, t)

		if fileMap[i].MinioStat.Key == "" {
			t.Fatalf("Unable to stat Minio object (%s)", v.fileNameWithPrefix)
		}

	}
}

func ValidateFileMetaData(t *testing.T) {
	for _, f := range fileMap {
		validateFileLSInfo(t, f)
		validateObjectMetaData(t, f)
		// validateContentType(t, f)
	}
}

func FindObjects(t *testing.T) {
	out, err := RunMC("find", mainTestBucket)
	fatalIfError(err, t)

	findList, err := parseFindJSONOutput(out)
	fatalIfError(err, t)

	for _, v := range fileMap {

		found := false
		for _, vv := range findList {
			if strings.HasSuffix(vv.Key, v.MinioLS.Key) {
				found = true
			}
		}

		if !found {
			t.Fatalf("File (%s) not found by 'find' command", v.MinioLS.Key)
		}
	}
}

func FindObjectsUsingName(t *testing.T) {
	for _, v := range fileMap {

		out, err := RunMC(
			"find",
			mainTestBucket,
			"--name",
			v.fileNameWithoutPath,
		)

		fatalIfError(err, t)
		info, err := parseFindSingleObjectJSONOutput(out)
		fatalIfError(err, t)
		if !strings.HasSuffix(info.Key, v.MinioLS.Key) {
			t.Fatalf("Invalid key (%s) when searching for (%s)", info.Key, v.MinioLS.Key)
		}

	}
}

func FindObjectsUsingNameAndFilteringForTxtType(t *testing.T) {
	out, err := RunMC(
		"find",
		mainTestBucket,
		"--name",
		"*.txt",
	)
	fatalIfError(err, t)

	findList, err := parseFindJSONOutput(out)
	fatalIfError(err, t)

	for _, v := range fileMap {
		if v.extension != ".txt" {
			continue
		}

		found := false
		for _, vv := range findList {
			if strings.HasSuffix(vv.Key, v.MinioLS.Key) {
				found = true
			}
		}

		if !found {
			t.Fatalf("File (%s) not found by 'find' command", v.MinioLS.Key)
		}
	}
}

func FindObjectsSmallerThan64Mebibytes(t *testing.T) {
	out, err := RunMC(
		"find",
		mainTestBucket,
		"--smaller",
		"64MB",
	)
	fatalIfError(err, t)

	findList, err := parseFindJSONOutput(out)
	fatalIfError(err, t)

	for _, v := range fileMap {
		if v.diskStat.Size() > GetMBSizeInBytes(64) {
			continue
		}

		found := false
		for _, vv := range findList {
			if strings.HasSuffix(vv.Key, v.MinioLS.Key) {
				found = true
			}
		}

		if !found {
			t.Fatalf("File (%s) not found by 'find' command", v.MinioLS.Key)
		}
	}
}

func FindObjectsLargerThan64Mebibytes(t *testing.T) {
	out, err := RunMC(
		"find",
		mainTestBucket,
		"--larger",
		"64MB",
	)
	fatalIfError(err, t)

	findList, err := parseFindJSONOutput(out)
	fatalIfError(err, t)

	for _, v := range fileMap {
		if v.diskStat.Size() < GetMBSizeInBytes(64) {
			continue
		}

		found := false
		for _, vv := range findList {
			if strings.HasSuffix(vv.Key, v.MinioLS.Key) {
				found = true
			}
		}

		if !found {
			t.Fatalf("File (%s) not found by 'find' command", v.MinioLS.Key)
		}
	}
}

func FindObjectsOlderThan1d(t *testing.T) {
	out, err := RunMC(
		"find",
		mainTestBucket,
		"--older-than",
		"1d",
	)
	fatalIfError(err, t)

	findList, err := parseFindJSONOutput(out)
	fatalIfError(err, t)

	if len(findList) > 0 {
		t.Fatalf("We should not have found any files which are older then 1 day")
	}
}

func FindObjectsNewerThen1d(t *testing.T) {
	out, err := RunMC(
		"find",
		mainTestBucket,
		"--newer-than",
		"1d",
	)
	fatalIfError(err, t)

	findList, err := parseFindJSONOutput(out)
	fatalIfError(err, t)

	for _, v := range fileMap {

		found := false
		for _, vv := range findList {
			if strings.HasSuffix(vv.Key, v.MinioLS.Key) {
				found = true
			}
		}

		if !found {
			t.Fatalf("File (%s) not found by 'find' command", v.MinioLS.Key)
		}
	}
}

func GetObjectsAndCompareMD5(t *testing.T) {
	for _, v := range fileMap {

		// make sure old downloads are not in our way
		_ = os.Remove(tempDir + "/" + v.fileNameWithoutPath + ".downloaded")

		_, err := RunMC(
			"cp",
			mainTestBucket+"/"+v.fileNameWithPrefix,
			tempDir+"/"+v.fileNameWithoutPath+".downloaded",
		)
		fatalIfError(err, t)

		downloadedFile, err := os.Open(
			tempDir + "/" + v.fileNameWithoutPath + ".downloaded",
		)
		fatalIfError(err, t)

		fileBytes, err := io.ReadAll(downloadedFile)
		fatalIfError(err, t)
		md5sum := GetMD5Sum(fileBytes)

		if v.md5Sum != md5sum {
			t.Fatalf(
				"The downloaded file md5sum is wrong: original-md5(%s) downloaded-md5(%s)",
				v.md5Sum,
				md5sum,
			)
		}
	}
}

func CreateBucketUsingInvalidSymbols(t *testing.T) {
	bucketNameMap := make(map[string]string)
	bucketNameMap["name-too-big"] = randomLargeString
	bucketNameMap["!"] = "symbol!"
	bucketNameMap["@"] = "symbol@"
	bucketNameMap["#"] = "symbol#"
	bucketNameMap["$"] = "symbol$"
	bucketNameMap["%"] = "symbol%"
	bucketNameMap["^"] = "symbol^"
	bucketNameMap["&"] = "symbol&"
	bucketNameMap["*"] = "symbol*"
	bucketNameMap["("] = "symbol("
	bucketNameMap[")"] = "symbol)"
	bucketNameMap["{"] = "symbol{"
	bucketNameMap["}"] = "symbol}"
	bucketNameMap["["] = "symbol["
	bucketNameMap["]"] = "symbol]"

	for _, v := range bucketNameMap {
		_, err := RunMC("mb", defaultAlias+"/"+v)
		if err == nil {
			t.Fatalf("We should not have been able to create a bucket with the name: %s", v)
		}
	}
}

func RemoveBucketThatDoesNotExist(t *testing.T) {
	randomID := uuid.NewString()
	out, _ := RunMC(
		"rb",
		defaultAlias+"/"+randomID,
	)
	errMSG, _ := parseSingleErrorMessageJSONOutput(out)
	validateErrorMSGValues(
		t,
		errMSG,
		"error",
		"Unable to validate",
		"does not exist",
	)
}

func RemoveBucketWithNameTooLong(t *testing.T) {
	randomID := uuid.NewString()
	out, _ := RunMC(
		"rb",
		defaultAlias+"/"+randomID+randomID,
	)
	errMSG, _ := parseSingleErrorMessageJSONOutput(out)
	validateErrorMSGValues(
		t,
		errMSG,
		"error",
		"Unable to validate",
		"Bucket name cannot be longer than 63 characters",
	)
}

func UploadToUnknownBucket(t *testing.T) {
	randomBucketID := uuid.NewString()
	parameters := append(
		[]string{},
		"cp",
		fileMap["1M"].diskFile.Name(),
		defaultAlias+"/"+randomBucketID+"-test-should-not-exist"+"/"+fileMap["1M"].fileNameWithoutPath,
	)

	_, err := RunMC(parameters...)
	if err == nil {
		t.Fatalf("We should not have been able to upload to bucket: %s", randomBucketID)
	}
}

func preRunCleanup() {
	for i := range tmpNameMap {
		_, _ = RunMC("rb", "--force", "--dangerous", defaultAlias+"/test-"+i)
	}
}

func postRunCleanup(t *testing.T) {
	var err error
	var berr error
	var out string

	err = os.RemoveAll(tempDir)
	if err != nil {
		fmt.Println(err)
	}

	for _, v := range bucketList {
		out, berr = RunMC("rb", "--force", "--dangerous", v)
		if berr != nil {
			fmt.Printf("Unable to remove bucket (%s) err: %s //  out: %s", v, berr, out)
		}
	}

	for _, v := range userList {
		_, _ = RunMC(
			"admin",
			"user",
			"remove",
			defaultAlias,
			v.Username,
		)
	}

	fatalIfError(berr, t)
	fatalIfError(err, t)
}

func validateFileLSInfo(t *testing.T, file *testFile) {
	if file.diskStat.Size() != int64(file.MinioLS.Size) {
		t.Fatalf(
			"File and minio object are not the same size - Object (%d) vs File (%d)",
			file.MinioLS.Size,
			file.diskStat.Size(),
		)
	}
	// if file.md5Sum != file.findOutput.Etag {
	// 	t.Fatalf("File and file.findOutput do not have the same md5Sum - Object (%s) vs File (%s)", file.findOutput.Etag, file.md5Sum)
	// }
	if file.storageClass != "" {
		if file.storageClass != file.MinioLS.StorageClass {
			t.Fatalf(
				"File and minio object do not have the same storage class - Object (%s) vs File (%s)",
				file.MinioLS.StorageClass,
				file.storageClass,
			)
		}
	} else {
		if file.MinioLS.StorageClass != "STANDARD" {
			t.Fatalf(
				"Minio object was expected to have storage class (STANDARD) but it was (%s)",
				file.MinioLS.StorageClass,
			)
		}
	}
}

func validateObjectMetaData(t *testing.T, file *testFile) {
	for i, v := range file.metaData {
		found := false

		for ii, vv := range file.MinioStat.Metadata {
			if metaPrefix+http.CanonicalHeaderKey(i) == ii {
				found = true
				if v != vv {
					fmt.Println("------------------------")
					fmt.Println("META CHECK")
					fmt.Println(file.MinioStat.Metadata)
					fmt.Println(file.metaData)
					fmt.Println("------------------------")
					t.Fatalf("Meta values are not the same v1(%s) v2(%s)", v, vv)
				}
			}
		}

		if !found {
			fmt.Println("------------------------")
			fmt.Println("META CHECK")
			fmt.Println(file.MinioStat.Metadata)
			fmt.Println(file.metaData)
			fmt.Println("------------------------")
			t.Fatalf("Meta tag(%s) not found", i)
		}

	}
}

// func validateContentType(t *testing.T, file *testFile) {
// 	value, ok := file.MinioStat.Metadata["Content-Type"]
// 	if !ok {
// 		t.Fatalf("File (%s) did not have a content type", file.fileNameWithPrefix)
// 		return
// 	}
//
// 	contentType := mime.TypeByExtension(file.extension)
// 	if contentType != value {
// 		log.Println(file)
// 		log.Println(file.MinioLS)
// 		log.Println(file.extension)
// 		log.Println(file.MinioStat)
// 		t.Fatalf("Content types on file (%s) do not match, extension(%s) File(%s) MinIO object(%s)", file.fileNameWithPrefix, file.extension, contentType, file.MinioStat.Metadata["Content-Type"])
// 	}
// }

func GetSource(skip int) (out string) {
	pc := make([]uintptr, 3) // at least 1 entry needed
	runtime.Callers(skip, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	sn := strings.Split(f.Name(), ".")
	var name string
	if sn[len(sn)-1] == "func1" {
		name = sn[len(sn)-2]
	} else {
		name = sn[len(sn)-1]
	}
	out = file + ":" + fmt.Sprint(line) + ":" + name
	return
}

func GetMD5Sum(data []byte) string {
	md5Writer := md5.New()
	md5Writer.Write(data)
	return fmt.Sprintf("%x", md5Writer.Sum(nil))
}

func curlFatalIfNoErrorTag(msg string, t *testing.T) {
	if !strings.Contains(msg, "<Error>") {
		fmt.Println(failIndicator)
		fmt.Println(msg)
		t.Fatal(msg)
	}
}

func fatalMsgOnly(msg string, t *testing.T) {
	fmt.Println(failIndicator)
	t.Fatal(msg)
}

func fatalIfNoErrorWMsg(err error, msg string, t *testing.T) {
	if err == nil {
		fmt.Println(failIndicator)
		fmt.Println(msg)
		t.Fatal(err)
	}
}

func fatalIfErrorWMsg(err error, msg string, t *testing.T) {
	if err != nil {
		fmt.Println(failIndicator)
		fmt.Println(msg)
		t.Fatal(err)
	}
}

func fatalIfError(err error, t *testing.T) {
	if err != nil {
		fmt.Println(failIndicator)
		t.Fatal(err)
	}
}

func parseFindJSONOutput(out string) (findList []*findMessage, err error) {
	findList = make([]*findMessage, 0)
	splitList := bytes.Split([]byte(out), []byte{10})

	for _, v := range splitList {
		if len(v) < 1 {
			continue
		}
		line := new(findMessage)
		err = json.Unmarshal(v, line)
		if err != nil {
			return
		}
		findList = append(findList, line)
	}

	if printRawOut {
		fmt.Println("FIND LIST ------------------------------")
		for _, v := range findList {
			fmt.Println(v)
		}
		fmt.Println(" ------------------------------")
	}
	return
}

func parseDUJSONOutput(out string) (duList []duMessage, err error) {
	duList = make([]duMessage, 0)
	splitList := bytes.Split([]byte(out), []byte{10})

	for _, v := range splitList {
		if len(v) < 1 {
			continue
		}
		line := duMessage{}
		err = json.Unmarshal(v, &line)
		if err != nil {
			return
		}
		duList = append(duList, line)
	}

	if printRawOut {
		fmt.Println("DU LIST ------------------------------")
		for _, v := range duList {
			fmt.Println(v)
		}
		fmt.Println(" ------------------------------")
	}
	return
}

func parseLSJSONOutput(out string) (lsList []contentMessage, err error) {
	lsList = make([]contentMessage, 0)
	splitList := bytes.Split([]byte(out), []byte{10})

	for _, v := range splitList {
		if len(v) < 1 {
			continue
		}
		line := contentMessage{}
		err = json.Unmarshal(v, &line)
		if err != nil {
			return
		}
		lsList = append(lsList, line)
	}

	if printRawOut {
		fmt.Println("LS LIST ------------------------------")
		for _, v := range lsList {
			fmt.Println(v)
		}
		fmt.Println(" ------------------------------")
	}
	return
}

func parseFindSingleObjectJSONOutput(out string) (findInfo contentMessage, err error) {
	err = json.Unmarshal([]byte(out), &findInfo)
	if err != nil {
		return
	}

	if printRawOut {
		fmt.Println("FIND SINGLE OBJECT ------------------------------")
		fmt.Println(findInfo)
		fmt.Println(" ------------------------------")
	}
	return
}

func parseStatSingleObjectJSONOutput(out string) (stat statMessage, err error) {
	err = json.Unmarshal([]byte(out), &stat)
	if err != nil {
		return
	}

	if printRawOut {
		fmt.Println("STAT ------------------------------")
		fmt.Println(stat)
		fmt.Println(" ------------------------------")
	}
	return
}

// We have to wrap the error output because the console
// printing mechanism for json marshals into an anonymous
// object before printing, see cmd/error.go line 70
type errorMessageWrapper struct {
	Error  errorMessage `json:"error"`
	Status string       `json:"status"`
}

func validateErrorMSGValues(
	t *testing.T,
	errMSG errorMessageWrapper,
	TypeToValidate string,
	MessageToValidate string,
	CauseToValidate string,
) {
	if TypeToValidate != "" {
		if !strings.Contains(errMSG.Error.Type, TypeToValidate) {
			t.Fatalf(
				"Expected error.Error.Type to contain (%s) - but got (%s)",
				TypeToValidate,
				errMSG.Error.Type,
			)
		}
	}
	if MessageToValidate != "" {
		if !strings.Contains(errMSG.Error.Message, MessageToValidate) {
			t.Fatalf(
				"Expected error.Error.Message to contain (%s) - but got (%s)",
				MessageToValidate,
				errMSG.Error.Message,
			)
		}
	}
	if CauseToValidate != "" {
		if !strings.Contains(errMSG.Error.Cause.Message, CauseToValidate) {
			t.Fatalf(
				"Expected error.Error.Cause.Message to contain (%s) - but got (%s)",
				CauseToValidate,
				errMSG.Error.Cause.Message,
			)
		}
	}
}

func parseUserMessageListOutput(out string) (users []*userMessage, err error) {
	users = make([]*userMessage, 0)
	splitList := bytes.Split([]byte(out), []byte{10})
	for _, v := range splitList {
		if len(v) < 1 {
			continue
		}
		msg := new(userMessage)
		err = json.Unmarshal(v, msg)
		if err != nil {
			return
		}
		users = append(users, msg)
	}

	if printRawOut {
		fmt.Println("USER LIST ------------------------------")
		for _, v := range users {
			fmt.Println(v)
		}
		fmt.Println(" ------------------------------")
	}

	return
}

func parseShareMessageFromJSONOutput(out string) (share *shareMessage, err error) {
	share = new(shareMessage)
	err = json.Unmarshal([]byte(out), share)
	return
}

func parseSingleErrorMessageJSONOutput(out string) (errMSG errorMessageWrapper, err error) {
	err = json.Unmarshal([]byte(out), &errMSG)
	if err != nil {
		return
	}

	fmt.Println("ERROR ------------------------------")
	fmt.Println(errMSG)
	fmt.Println(" ------------------------------")
	return
}

func parseSingleODMessageJSONOutput(out string) (odMSG odMessage, err error) {
	err = json.Unmarshal([]byte(out), &odMSG)
	if err != nil {
		return
	}

	return
}

func parseSingleAccountStatJSONOutput(out string) (stat accountStat, err error) {
	err = json.Unmarshal([]byte(out), &stat)
	if err != nil {
		return
	}

	return
}

func parseSingleCPMessageJSONOutput(out string) (cpMSG copyMessage, err error) {
	err = json.Unmarshal([]byte(out), &cpMSG)
	if err != nil {
		return
	}

	return
}

type newTestFile struct {
	tag          string // The tag used to identify the file inside the FileMap. This tag is also used in the objects name.
	prefix       string // Prefix for the object name ( not including the object name itself)
	extension    string
	storageClass string
	sizeInMBS    int
	// uploadShouldFail bool
	metaData map[string]string
	tags     map[string]string

	addToGlobalFileMap bool
	// sub directory path to place the file in
	// tempDir+/+subDir
	subDir string
}

type testFile struct {
	newTestFile

	// File on disk
	diskFile *os.File
	// File info on disk
	diskStat os.FileInfo
	// md5sum at the time of creation
	md5Sum string
	// File name without full path
	fileNameWithoutPath string
	// File name with assigned prefix
	fileNameWithPrefix string

	// These field are not automatically populated unless
	// the file is created at the initialization phase of
	// the test suite: testsThatDependOnOneAnother()
	// Minio mc stat output
	MinioStat statMessage
	// Minio mc ls output
	MinioLS contentMessage
}

func (f *testFile) String() (out string) {
	out = fmt.Sprintf(
		"Size: %d || Name: %s || md5Sum: %s",
		f.diskStat.Size(),
		f.fileNameWithoutPath,
		f.md5Sum,
	)
	return
}

func createFile(nf newTestFile) (newTestFile *testFile) {
	var newFile *os.File
	var err error
	if nf.subDir != "" {

		err = os.MkdirAll(
			tempDir+string(os.PathSeparator)+nf.subDir,
			0o755)
		if err != nil {
			log.Println("Could not make additional dir:", err)
			os.Exit(1)
		}

		newFile, err = os.CreateTemp(
			tempDir+string(os.PathSeparator)+nf.subDir,
			nf.tag+"-*"+nf.extension,
		)

	} else {
		newFile, err = os.CreateTemp(tempDir, nf.tag+"-*"+nf.extension)
	}

	if err != nil {
		log.Println("Could not make file:", err)
		os.Exit(1)
	}

	md5Writer := md5.New()
	for i := 0; i < nf.sizeInMBS; i++ {
		n, err := newFile.Write(oneMBSlice[:])
		mn, merr := md5Writer.Write(oneMBSlice[:])
		if err != nil || merr != nil {
			log.Println(err)
			log.Println(merr)
			return nil
		}
		if n != len(oneMBSlice) {
			log.Println("Did not write 1MB to file")
			return nil
		}
		if mn != len(oneMBSlice) {
			log.Println("Did not write 1MB to md5sum writer")
			return nil
		}
	}
	splitName := strings.Split(newFile.Name(), string(os.PathSeparator))
	fileNameWithoutPath := splitName[len(splitName)-1]
	md5sum := fmt.Sprintf("%x", md5Writer.Sum(nil))
	stats, err := newFile.Stat()
	if err != nil {
		return nil
	}
	newTestFile = &testFile{
		md5Sum:              md5sum,
		fileNameWithoutPath: fileNameWithoutPath,
		diskFile:            newFile,
		diskStat:            stats,
	}

	newTestFile.tag = nf.tag
	newTestFile.metaData = nf.metaData
	newTestFile.storageClass = nf.storageClass
	newTestFile.sizeInMBS = nf.sizeInMBS
	newTestFile.tags = nf.tags
	newTestFile.prefix = nf.prefix
	newTestFile.extension = nf.extension

	if nf.prefix != "" {
		newTestFile.fileNameWithPrefix = nf.prefix + "/" + fileNameWithoutPath
	} else {
		newTestFile.fileNameWithPrefix = fileNameWithoutPath
	}
	if nf.addToGlobalFileMap {
		fileMap[nf.tag] = newTestFile
	}
	return newTestFile
}

func BuildCLI() error {
	wd, _ := os.Getwd()
	fmt.Println("WORKING DIR:", wd)
	fmt.Println("go build -o", mcCmd, buildPath)
	os.Remove(mcCmd)
	out, err := exec.Command("go", "build", "-o", mcCmd, buildPath).CombinedOutput()
	if err != nil {
		log.Println("BUILD OUT:", string(out))
		log.Println(err)
		panic(err)
	}
	err = os.Chmod(mcCmd, 0o777)
	if err != nil {
		panic(err)
	}
	return nil
}

func RunMC(parameters ...string) (out string, err error) {
	var outBytes []byte
	var outErr error

	fmt.Println("")
	fmt.Println(time.Now().Format("2006-01-02T15:04:05.000"), "||", GetSource(3))
	fmt.Println(mcCmd, strings.Join(preCmdParameters, " "), strings.Join(parameters, " "))

	outBytes, outErr = exec.Command(mcCmd, append(preCmdParameters, parameters...)...).CombinedOutput()
	if printRawOut {
		fmt.Println(string(outBytes))
	}
	out = string(outBytes)
	err = outErr
	return
}

func RunCommand(cmd string, parameters ...string) (out string, err error) {
	fmt.Println("")
	fmt.Println(time.Now().Format("2006-01-02T15:04:05.000"), "||", GetSource(3))
	fmt.Println(cmd, strings.Join(parameters, " "))
	var outBytes []byte
	var outErr error

	outBytes, outErr = exec.Command(cmd, parameters...).CombinedOutput()
	if printRawOut {
		fmt.Println(string(outBytes))
	}
	out = string(outBytes)
	err = outErr
	return
}
