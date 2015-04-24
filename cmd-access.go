package main

import (
	"fmt"
	"time"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

func runAccessCmd(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "access", 1) // last argument is exit code
	}

	config, err := getMcConfig()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("mc: loading config file failed with following reason: [%s]\n", iodine.ToError(err))
	}

	targetURLConfigMap := make(map[string]*hostConfig)
	targetURLs, err := getURLs(ctx.Args(), config.Aliases)
	if err != nil {
		switch e := iodine.ToError(err).(type) {
		case errUnsupportedScheme:
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("mc: reading URL [%s] failed with following reason: [%s]\n", e.url, e)
		default:
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("mc: reading URLs failed with following reason: [%s]\n", e)
		}
	}
	acl := bucketACL(ctx.Args().First())
	if !acl.isValidBucketACL() {
		log.Debug.Println(iodine.New(errInvalidACL{acl: acl.String()}, nil))
		console.Fatalf("Access type [%s] is not supported. Valid types are [private, public, readonly].\n", acl)
	}
	targetURLs = targetURLs[1:] // 1 or more target URLs
	for _, targetURL := range targetURLs {
		targetConfig, err := getHostConfig(targetURL)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("Unable to read configuration for host [%s]. Reason: [%s].\n", targetURL, iodine.ToError(err))
		}
		targetURLConfigMap[targetURL] = targetConfig
	}
	for targetURL, targetConfig := range targetURLConfigMap {
		errorMsg, err := doUpdateAccessCmd(mcClientManager{}, targetURL, acl.String(), targetConfig, globalDebugFlag)
		err = iodine.New(err, nil)
		if err != nil {
			if errorMsg == "" {
				errorMsg = "Empty error message.  Please rerun this command with --debug and file a bug report."
			}
			log.Debug.Println(err)
			console.Errorf("%s", errorMsg)
		}
	}
}

func doUpdateAccessCmd(manager clientManager, targetURL, targetACL string, targetConfig *hostConfig, debug bool) (string, error) {
	var err error
	var clnt client.Client
	clnt, err = manager.getNewClient(targetURL, targetConfig, debug)
	if err != nil {
		err := iodine.New(err, nil)
		msg := fmt.Sprintf("Unable to initialize client for [%s]. Reason: [%s].\n",
			targetURL, iodine.ToError(err))
		return msg, err
	}
	return doUpdateAccess(clnt, targetURL, targetACL)
}

func doUpdateAccess(clnt client.Client, targetURL, targetACL string) (string, error) {
	err := clnt.PutBucket(targetACL)
	if err != nil && isValidRetry(err) {
		console.Infof("Retrying ...")
	}
	for i := 0; i < globalMaxRetryFlag && err != nil && isValidRetry(err); i++ {
		err = clnt.PutBucket(targetACL)
		console.Errorf(" %d", i)
		// Progressively longer delays
		time.Sleep(time.Duration(i*i) * time.Second)
	}
	if err != nil {
		err := iodine.New(err, nil)
		msg := fmt.Sprintf("Failed to add bucket access policy for URL [%s]. Reason: [%s].\n", targetURL, iodine.ToError(err))
		return msg, err
	}
	return "", nil
}
