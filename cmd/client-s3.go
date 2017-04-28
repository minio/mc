/*
 * Minio Client (C) 2015, 2016, 2017 Minio, Inc.
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

package cmd

import (
	"crypto/tls"
	"errors"
	"hash/fnv"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"io/ioutil"

	"github.com/minio/mc/pkg/httptracer"
	"github.com/minio/minio-go"
	"github.com/minio/minio-go/pkg/policy"
	"github.com/minio/minio-go/pkg/s3utils"
	"github.com/minio/minio/pkg/probe"
)

// S3 client
type s3Client struct {
	mutex        *sync.Mutex
	targetURL    *clientURL
	api          *minio.Client
	virtualStyle bool
}

const (
	amazonHostName            = "s3.amazonaws.com"
	amazonHostNameAccelerated = "s3-accelerate.amazonaws.com"

	googleHostName = "storage.googleapis.com"
)

// newFactory encloses New function with client cache.
func newFactory() func(config *Config) (Client, *probe.Error) {
	clientCache := make(map[uint32]*minio.Client)
	mutex := &sync.Mutex{}

	// Return New function.
	return func(config *Config) (Client, *probe.Error) {
		// Creates a parsed URL.
		targetURL := newClientURL(config.HostURL)
		// By default enable HTTPs.
		useTLS := true
		if targetURL.Scheme == "http" {
			useTLS = false
		}

		// Instantiate s3
		s3Clnt := &s3Client{}
		// Allocate a new mutex.
		s3Clnt.mutex = new(sync.Mutex)
		// Save the target URL.
		s3Clnt.targetURL = targetURL

		// Save if target supports virtual host style.
		hostName := targetURL.Host
		s3Clnt.virtualStyle = isVirtualHostStyle(hostName)
		isS3AcceleratedEndpoint := isAmazonAccelerated(hostName)

		if s3Clnt.virtualStyle {
			// If Amazon URL replace it with 's3.amazonaws.com'
			if isAmazon(hostName) || isAmazonAccelerated(hostName) {
				hostName = amazonHostName
			}

			// If Google URL replace it with 'storage.googleapis.com'
			if isGoogle(hostName) {
				hostName = googleHostName
			}
		}

		// Generate a hash out of s3Conf.
		confHash := fnv.New32a()
		confHash.Write([]byte(hostName + config.AccessKey + config.SecretKey))
		confSum := confHash.Sum32()

		// Lookup previous cache by hash.
		mutex.Lock()
		defer mutex.Unlock()
		var api *minio.Client
		var found bool
		if api, found = clientCache[confSum]; !found {
			// Not found. Instantiate a new minio
			var e error
			if strings.ToUpper(config.Signature) == "S3V2" {
				// if Signature version '2' use NewV2 directly.
				api, e = minio.NewV2(hostName, config.AccessKey, config.SecretKey, useTLS)
			} else {
				// if Signature version '4' use NewV4 directly.
				api, e = minio.NewV4(hostName, config.AccessKey, config.SecretKey, useTLS)
			}
			if e != nil {
				return nil, probe.NewError(e)
			}

			// Keep TLS config.
			tlsConfig := &tls.Config{RootCAs: globalRootCAs}
			if config.Insecure {
				tlsConfig.InsecureSkipVerify = true
			}

			var transport http.RoundTripper = &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				TLSClientConfig:       tlsConfig,
			}

			if config.Debug {
				if strings.EqualFold(config.Signature, "S3v4") {
					transport = httptracer.GetNewTraceTransport(newTraceV4(), transport)
				} else if strings.EqualFold(config.Signature, "S3v2") {
					transport = httptracer.GetNewTraceTransport(newTraceV2(), transport)
				}
				// Set custom transport.
			}

			// Set custom transport.
			api.SetCustomTransport(transport)

			// If Amazon Accelerated URL is requested enable it.
			if isS3AcceleratedEndpoint {
				api.SetS3TransferAccelerate(amazonHostNameAccelerated)
			}

			// Set app info.
			api.SetAppInfo(config.AppName, config.AppVersion)

			// Cache the new minio client with hash of config as key.
			clientCache[confSum] = api
		}

		// Store the new api object.
		s3Clnt.api = api

		return s3Clnt, nil
	}
}

// s3New returns an initialized s3Client structure. If debug is enabled,
// it also enables an internal trace transport.
var s3New = newFactory()

// GetURL get url.
func (c *s3Client) GetURL() clientURL {
	return *c.targetURL
}

// Add bucket notification
func (c *s3Client) AddNotificationConfig(arn string, events []string, prefix, suffix string) *probe.Error {
	bucket, _ := c.url2BucketAndObject()
	if err := isValidBucketName(bucket); err != nil {
		return err
	}

	// Validate total fields in ARN.
	fields := strings.Split(arn, ":")
	if len(fields) != 6 {
		return errInvalidArgument()
	}

	// Get any enabled notification.
	mb, e := c.api.GetBucketNotification(bucket)
	if e != nil {
		return probe.NewError(e)
	}

	accountArn := minio.NewArn(fields[1], fields[2], fields[3], fields[4], fields[5])
	nc := minio.NewNotificationConfig(accountArn)

	// Configure events
	for _, event := range events {
		switch event {
		case "put":
			nc.AddEvents(minio.ObjectCreatedAll)
		case "delete":
			nc.AddEvents(minio.ObjectRemovedAll)
		case "get":
			nc.AddEvents(minio.ObjectAccessedAll)
		default:
			return errInvalidArgument().Trace(events...)
		}
	}
	if prefix != "" {
		nc.AddFilterPrefix(prefix)
	}
	if suffix != "" {
		nc.AddFilterSuffix(suffix)
	}

	switch fields[2] {
	case "sns":
		mb.AddTopic(nc)
	case "sqs":
		mb.AddQueue(nc)
	case "lambda":
		mb.AddLambda(nc)
	default:
		return errInvalidArgument().Trace(fields[2])
	}

	// Set the new bucket configuration
	if err := c.api.SetBucketNotification(bucket, mb); err != nil {
		return probe.NewError(err)
	}
	return nil
}

// Remove bucket notification
func (c *s3Client) RemoveNotificationConfig(arn string) *probe.Error {
	bucket, _ := c.url2BucketAndObject()
	if err := isValidBucketName(bucket); err != nil {
		return err
	}

	// Remove all notification configs if arn is empty
	if arn == "" {
		if err := c.api.RemoveAllBucketNotification(bucket); err != nil {
			return probe.NewError(err)
		}
		return nil
	}

	mb, e := c.api.GetBucketNotification(bucket)
	if e != nil {
		return probe.NewError(e)
	}

	fields := strings.Split(arn, ":")
	if len(fields) != 6 {
		return errInvalidArgument().Trace(fields...)
	}
	accountArn := minio.NewArn(fields[1], fields[2], fields[3], fields[4], fields[5])

	switch fields[2] {
	case "sns":
		mb.RemoveTopicByArn(accountArn)
	case "sqs":
		mb.RemoveQueueByArn(accountArn)
	case "lambda":
		mb.RemoveLambdaByArn(accountArn)
	default:
		return errInvalidArgument().Trace(fields[2])
	}

	// Set the new bucket configuration
	if e := c.api.SetBucketNotification(bucket, mb); e != nil {
		return probe.NewError(e)
	}
	return nil
}

type notificationConfig struct {
	ID     string   `json:"id"`
	Arn    string   `json:"arn"`
	Events []string `json:"events"`
	Prefix string   `json:"prefix"`
	Suffix string   `json:"suffix"`
}

// List notification configs
func (c *s3Client) ListNotificationConfigs(arn string) ([]notificationConfig, *probe.Error) {
	var configs []notificationConfig
	bucket, _ := c.url2BucketAndObject()
	if err := isValidBucketName(bucket); err != nil {
		return nil, err
	}

	mb, e := c.api.GetBucketNotification(bucket)
	if e != nil {
		return nil, probe.NewError(e)
	}

	// Generate pretty event names from event types
	prettyEventNames := func(eventsTypes []minio.NotificationEventType) []string {
		var result []string
		for _, eventType := range eventsTypes {
			result = append(result, string(eventType))
		}
		return result
	}

	getFilters := func(config minio.NotificationConfig) (prefix, suffix string) {
		if config.Filter == nil {
			return
		}
		for _, filter := range config.Filter.S3Key.FilterRules {
			if strings.ToLower(filter.Name) == "prefix" {
				prefix = filter.Value
			}
			if strings.ToLower(filter.Name) == "suffix" {
				suffix = filter.Value
			}

		}
		return prefix, suffix
	}

	for _, config := range mb.TopicConfigs {
		if arn != "" && config.Topic != arn {
			continue
		}
		prefix, suffix := getFilters(config.NotificationConfig)
		configs = append(configs, notificationConfig{ID: config.ID,
			Arn:    config.Topic,
			Events: prettyEventNames(config.Events),
			Prefix: prefix,
			Suffix: suffix})
	}

	for _, config := range mb.QueueConfigs {
		if arn != "" && config.Queue != arn {
			continue
		}
		prefix, suffix := getFilters(config.NotificationConfig)
		configs = append(configs, notificationConfig{ID: config.ID,
			Arn:    config.Queue,
			Events: prettyEventNames(config.Events),
			Prefix: prefix,
			Suffix: suffix})
	}

	for _, config := range mb.LambdaConfigs {
		if arn != "" && config.Lambda != arn {
			continue
		}
		prefix, suffix := getFilters(config.NotificationConfig)
		configs = append(configs, notificationConfig{ID: config.ID,
			Arn:    config.Lambda,
			Events: prettyEventNames(config.Events),
			Prefix: prefix,
			Suffix: suffix})
	}

	return configs, nil
}

// Start watching on all bucket events for a given account ID.
func (c *s3Client) Watch(params watchParams) (*watchObject, *probe.Error) {
	eventChan := make(chan EventInfo)
	errorChan := make(chan *probe.Error)
	doneChan := make(chan bool)

	// Extract bucket and object.
	bucket, object := c.url2BucketAndObject()
	if err := isValidBucketName(bucket); err != nil {
		return nil, err
	}

	// Flag set to set the notification.
	var events []string
	for _, event := range params.events {
		switch event {
		case "put":
			events = append(events, string(minio.ObjectCreatedAll))
		case "delete":
			events = append(events, string(minio.ObjectRemovedAll))
		case "get":
			events = append(events, string(minio.ObjectAccessedAll))
		default:
			return nil, errInvalidArgument().Trace(event)
		}
	}
	if object != "" && params.prefix != "" {
		return nil, errInvalidArgument().Trace(params.prefix, object)
	}
	if object != "" && params.prefix == "" {
		params.prefix = object
	}

	doneCh := make(chan struct{})

	// wait for doneChan to close the other channels
	go func() {
		<-doneChan

		close(doneCh)
		close(eventChan)
		close(errorChan)
	}()

	// Start listening on all bucket events.
	eventsCh := c.api.ListenBucketNotification(bucket, params.prefix, params.suffix, events, doneCh)

	wo := &watchObject{
		eventInfoChan: eventChan,
		errorChan:     errorChan,
		doneChan:      doneChan,
	}

	// wait for events to occur and sent them through the eventChan and errorChan
	go func() {
		defer wo.Close()
		for notificationInfo := range eventsCh {
			if notificationInfo.Err != nil {
				if nErr, ok := notificationInfo.Err.(minio.ErrorResponse); ok && nErr.Code == "APINotSupported" {
					errorChan <- probe.NewError(APINotImplemented{
						API:     "Watch",
						APIType: c.targetURL.Scheme + "://" + c.targetURL.Host,
					})
					return
				}
				errorChan <- probe.NewError(notificationInfo.Err)
			}

			for _, record := range notificationInfo.Records {
				bucketName := record.S3.Bucket.Name
				key, e := url.QueryUnescape(record.S3.Object.Key)
				if e != nil {
					errorChan <- probe.NewError(e)
					continue
				}

				u := *c.targetURL
				u.Path = path.Join(string(u.Separator), bucketName, key)
				if strings.HasPrefix(record.EventName, "s3:ObjectCreated:") {
					eventChan <- EventInfo{
						Time:      record.EventTime,
						Size:      record.S3.Object.Size,
						Path:      u.String(),
						Client:    c,
						Type:      EventCreate,
						Host:      record.Source.Host,
						Port:      record.Source.Port,
						UserAgent: record.Source.UserAgent,
					}

				} else if strings.HasPrefix(record.EventName, "s3:ObjectRemoved:") {
					eventChan <- EventInfo{
						Time:      record.EventTime,
						Path:      u.String(),
						Client:    c,
						Type:      EventRemove,
						Host:      record.Source.Host,
						Port:      record.Source.Port,
						UserAgent: record.Source.UserAgent,
					}
				} else if record.EventName == minio.ObjectAccessedGet {
					eventChan <- EventInfo{
						Time:      record.EventTime,
						Size:      record.S3.Object.Size,
						Path:      u.String(),
						Client:    c,
						Type:      EventAccessedRead,
						Host:      record.Source.Host,
						Port:      record.Source.Port,
						UserAgent: record.Source.UserAgent,
					}
				} else if record.EventName == minio.ObjectAccessedHead {
					eventChan <- EventInfo{
						Time:      record.EventTime,
						Size:      record.S3.Object.Size,
						Path:      u.String(),
						Client:    c,
						Type:      EventAccessedStat,
						Host:      record.Source.Host,
						Port:      record.Source.Port,
						UserAgent: record.Source.UserAgent,
					}
				}
			}
		}
	}()

	return wo, nil
}

// Get - get object with metadata.
func (c *s3Client) Get() (io.Reader, map[string][]string, *probe.Error) {
	bucket, object := c.url2BucketAndObject()
	reader, e := c.api.GetObject(bucket, object)
	if e != nil {
		errResponse := minio.ToErrorResponse(e)
		if errResponse.Code == "NoSuchBucket" {
			return nil, nil, probe.NewError(BucketDoesNotExist{
				Bucket: bucket,
			})
		}
		if errResponse.Code == "InvalidBucketName" {
			return nil, nil, probe.NewError(BucketInvalid{
				Bucket: bucket,
			})
		}
		if errResponse.Code == "NoSuchKey" || errResponse.Code == "InvalidArgument" {
			return nil, nil, probe.NewError(ObjectMissing{})
		}
		return nil, nil, probe.NewError(e)
	}
	objInfo, e := reader.Stat()
	if e != nil {
		errResponse := minio.ToErrorResponse(e)
		if errResponse.Code == "AccessDenied" {
			return nil, nil, probe.NewError(PathInsufficientPermission{Path: c.targetURL.String()})
		}
		if errResponse.Code == "NoSuchKey" || errResponse.Code == "InvalidArgument" {
			return nil, nil, probe.NewError(ObjectMissing{})
		}
		return nil, nil, probe.NewError(e)
	}
	metadata := objInfo.Metadata
	metadata.Set("Content-Type", objInfo.ContentType)
	return reader, metadata, nil
}

// Copy - copy object
func (c *s3Client) Copy(source string, size int64, progress io.Reader) *probe.Error {
	bucket, object := c.url2BucketAndObject()
	if bucket == "" {
		return probe.NewError(BucketNameEmpty{})
	}
	// Empty copy conditions
	copyConds := minio.NewCopyConditions()
	e := c.api.CopyObject(bucket, object, source, copyConds)
	if e != nil {
		errResponse := minio.ToErrorResponse(e)
		if errResponse.Code == "AccessDenied" {
			return probe.NewError(PathInsufficientPermission{
				Path: c.targetURL.String(),
			})
		}
		if errResponse.Code == "NoSuchBucket" {
			return probe.NewError(BucketDoesNotExist{
				Bucket: bucket,
			})
		}
		if errResponse.Code == "InvalidBucketName" {
			return probe.NewError(BucketInvalid{
				Bucket: bucket,
			})
		}
		if errResponse.Code == "NoSuchKey" || errResponse.Code == "InvalidArgument" {
			return probe.NewError(ObjectMissing{})
		}
		return probe.NewError(e)
	}
	// Successful copy update progress bar if there is one.
	if progress != nil {
		if _, e := io.CopyN(ioutil.Discard, progress, size); e != nil {
			return probe.NewError(e)
		}
	}
	return nil
}

// Put - upload an object with custom metadata.
func (c *s3Client) Put(reader io.Reader, size int64, metadata map[string][]string, progress io.Reader) (int64, *probe.Error) {
	bucket, object := c.url2BucketAndObject()
	_, ok := metadata["Content-Type"]
	if !ok {
		// Set content-type if not specified.
		metadata["Content-Type"] = []string{"application/octet-stream"}
	}
	if bucket == "" {
		return 0, probe.NewError(BucketNameEmpty{})
	}
	n, e := c.api.PutObjectWithMetadata(bucket, object, reader, metadata, progress)
	if e != nil {
		errResponse := minio.ToErrorResponse(e)
		if errResponse.Code == "UnexpectedEOF" || e == io.EOF {
			return n, probe.NewError(UnexpectedEOF{
				TotalSize:    size,
				TotalWritten: n,
			})
		}
		if errResponse.Code == "AccessDenied" {
			return n, probe.NewError(PathInsufficientPermission{
				Path: c.targetURL.String(),
			})
		}
		if errResponse.Code == "MethodNotAllowed" {
			return n, probe.NewError(ObjectAlreadyExists{
				Object: object,
			})
		}
		if errResponse.Code == "XMinioObjectExistsAsDirectory" {
			return n, probe.NewError(ObjectAlreadyExistsAsDirectory{
				Object: object,
			})
		}
		if errResponse.Code == "NoSuchBucket" {
			return n, probe.NewError(BucketDoesNotExist{
				Bucket: bucket,
			})
		}
		if errResponse.Code == "InvalidBucketName" {
			return n, probe.NewError(BucketInvalid{
				Bucket: bucket,
			})
		}
		if errResponse.Code == "NoSuchKey" || errResponse.Code == "InvalidArgument" {
			return n, probe.NewError(ObjectMissing{})
		}
		return n, probe.NewError(e)
	}
	return n, nil
}

// Remove incomplete uploads.
func (c *s3Client) removeIncompleteObjects(bucket string, objectsCh <-chan string) <-chan minio.RemoveObjectError {
	removeObjectErrorCh := make(chan minio.RemoveObjectError)

	// Goroutine reads from objectsCh and sends error to removeObjectErrorCh if any.
	go func() {
		defer close(removeObjectErrorCh)

		for object := range objectsCh {
			if err := c.api.RemoveIncompleteUpload(bucket, object); err != nil {
				removeObjectErrorCh <- minio.RemoveObjectError{ObjectName: object, Err: err}
			}
		}
	}()

	return removeObjectErrorCh
}

// Remove - remove object or bucket.
func (c *s3Client) Remove(isIncomplete bool, contentCh <-chan *clientContent) <-chan *probe.Error {
	bucket, _ := c.url2BucketAndObject()

	errorCh := make(chan *probe.Error)
	var bucketContent *clientContent

	// Goroutine
	// 1. calls removeIncompleteObjects() for incomplete uploads
	//    or minio-go.RemoveObjects().
	// 2. executes another Goroutine to copy contentCh to objectsCh.
	// 3. reads statusCh and copies to errorCh.
	go func() {
		defer close(errorCh)

		objectsCh := make(chan string)
		var statusCh <-chan minio.RemoveObjectError
		if isIncomplete {
			statusCh = c.removeIncompleteObjects(bucket, objectsCh)
		} else {
			statusCh = c.api.RemoveObjects(bucket, objectsCh)
		}

		// doneCh to control below Goroutine.
		doneCh := make(chan struct{})
		defer close(doneCh)

		// Goroutine reads contentCh and copies to objectsCh.
		go func() {
			defer close(objectsCh)

			for {
				// Read from contentCh or doneCh.  If doneCh is read, exit the function.
				var content *clientContent
				var ok bool
				select {
				case content, ok = <-contentCh:
					if !ok {
						// Closed channel.
						return
					}
				case <-doneCh:
					return
				}

				// Convert content.URL.Path to objectName for objectsCh.
				_, objectName := c.splitPath(content.URL.Path)

				// Currently only supported hosts with virtual style
				// are Amazon S3 and Google Cloud Storage.
				// which also support objects with "/" as delimiter.
				// Skip trimming "/" and let the server reply error
				// if any.
				if !c.virtualStyle {
					objectName = strings.TrimSuffix(objectName, string(c.targetURL.Separator))
				}

				// As object name is empty, we need to remove the bucket as well.
				if objectName == "" {
					bucketContent = content
					continue
				}

				// Write to objectsCh or read doneCh. If doneCh is read, exit the function.
				select {
				case objectsCh <- objectName:
				case <-doneCh:
					return
				}
			}
		}()

		// Read statusCh and write to errorCh.
		for removeStatus := range statusCh {
			errorCh <- probe.NewError(removeStatus.Err)
		}

		// Remove bucket for regular objects.
		if bucketContent != nil && !isIncomplete {
			if err := c.api.RemoveBucket(bucket); err != nil {
				errorCh <- probe.NewError(err)
			}
		}
	}()

	return errorCh
}

// We support '.' with bucket names but we fallback to using path
// style requests instead for such buckets
var validBucketName = regexp.MustCompile(`^[a-z0-9][a-z0-9\.\-]{1,61}[a-z0-9]$`)

// isValidBucketName - verify bucket name in accordance with
//  - http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingBucket.html
func isValidBucketName(bucketName string) *probe.Error {
	if strings.TrimSpace(bucketName) == "" {
		return probe.NewError(errors.New("Bucket name cannot be empty"))
	}
	if len(bucketName) < 3 || len(bucketName) > 63 {
		return probe.NewError(errors.New("Bucket name should be more than 3 characters and less than 64 characters"))
	}
	if !validBucketName.MatchString(bucketName) {
		return probe.NewError(errors.New("Bucket names can only contain lowercase alpha characters `a-z`, numbers '0-9', or '-'. First/last character cannot be a '-'"))
	}
	return nil
}

// MakeBucket - make a new bucket.
func (c *s3Client) MakeBucket(region string, ignoreExisting bool) *probe.Error {
	bucket, object := c.url2BucketAndObject()
	if object != "" {
		return probe.NewError(BucketNameTopLevel{})
	}
	if err := isValidBucketName(bucket); err != nil {
		return err.Trace(bucket)
	}
	e := c.api.MakeBucket(bucket, region)
	if e != nil {
		// Ignore bucket already existing error when ignoreExisting flag is enabled
		if ignoreExisting {
			switch minio.ToErrorResponse(e).Code {
			case "BucketAlreadyOwnedByYou":
				fallthrough
			case "BucketAlreadyExists":
				return nil
			}
		}
		return probe.NewError(e)
	}
	return nil
}

// GetAccessRules - get configured policies from the server
func (c *s3Client) GetAccessRules() (map[string]string, *probe.Error) {
	bucket, object := c.url2BucketAndObject()
	if bucket == "" {
		return map[string]string{}, probe.NewError(BucketNameEmpty{})
	}
	policies := map[string]string{}
	policyRules, err := c.api.ListBucketPolicies(bucket, object)
	if err != nil {
		return nil, probe.NewError(err)
	}
	// Hide policy data structure at this level
	for k, v := range policyRules {
		policies[k] = string(v)
	}
	return policies, nil
}

// GetAccess get access policy permissions.
func (c *s3Client) GetAccess() (string, *probe.Error) {
	bucket, object := c.url2BucketAndObject()
	if bucket == "" {
		return "", probe.NewError(BucketNameEmpty{})
	}
	bucketPolicy, e := c.api.GetBucketPolicy(bucket, object)
	if e != nil {
		return "", probe.NewError(e)
	}
	return string(bucketPolicy), nil
}

// SetAccess set access policy permissions.
func (c *s3Client) SetAccess(bucketPolicy string) *probe.Error {
	bucket, object := c.url2BucketAndObject()
	if bucket == "" {
		return probe.NewError(BucketNameEmpty{})
	}
	e := c.api.SetBucketPolicy(bucket, object, policy.BucketPolicy(bucketPolicy))
	if e != nil {
		return probe.NewError(e)
	}
	return nil
}

// listObjectWrapper - select ObjectList version depending on the target hostname
func (c *s3Client) listObjectWrapper(bucket, object string, isRecursive bool, doneCh chan struct{}) <-chan minio.ObjectInfo {
	if isAmazon(c.targetURL.Host) || isAmazonAccelerated(c.targetURL.Host) {
		return c.api.ListObjectsV2(bucket, object, isRecursive, doneCh)
	}
	return c.api.ListObjects(bucket, object, isRecursive, doneCh)
}

// Stat - send a 'HEAD' on a bucket or object to fetch its metadata.
func (c *s3Client) Stat(isIncomplete bool) (*clientContent, *probe.Error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	bucket, object := c.url2BucketAndObject()
	// Bucket name cannot be empty, stat on URL has no meaning.
	if bucket == "" {
		return nil, probe.NewError(BucketNameEmpty{})
	}

	if object == "" {
		exists, e := c.api.BucketExists(bucket)
		if e != nil {
			return nil, probe.NewError(e)
		}
		if !exists {
			return nil, probe.NewError(BucketDoesNotExist{Bucket: bucket})
		}
		bucketMetadata := &clientContent{}
		bucketMetadata.URL = *c.targetURL
		bucketMetadata.Type = os.ModeDir

		return bucketMetadata, nil
	}

	// Remove trailing slashes needed for the following ListObjects call.
	// In addition, Stat() will be as smart as the client fs version and will
	// facilitate the work of the upper layers
	object = strings.TrimRight(object, string(c.targetURL.Separator))
	nonRecursive := false
	objectMetadata := &clientContent{}

	// If the request is for incomplete upload stat, handle it here.
	if isIncomplete {
		for objectMultipartInfo := range c.api.ListIncompleteUploads(bucket, object, nonRecursive, nil) {
			if objectMultipartInfo.Err != nil {
				return nil, probe.NewError(objectMultipartInfo.Err)
			}

			if objectMultipartInfo.Key == object {
				objectMetadata.URL = *c.targetURL
				objectMetadata.Time = objectMultipartInfo.Initiated
				objectMetadata.Size = objectMultipartInfo.Size
				objectMetadata.Type = os.FileMode(0664)
				return objectMetadata, nil
			}

			if strings.HasSuffix(objectMultipartInfo.Key, string(c.targetURL.Separator)) {
				objectMetadata.URL = *c.targetURL
				objectMetadata.Type = os.ModeDir
				return objectMetadata, nil
			}
		}
		return nil, probe.NewError(ObjectMissing{})
	}

	for objectStat := range c.listObjectWrapper(bucket, object, nonRecursive, nil) {
		if objectStat.Err != nil {
			return nil, probe.NewError(objectStat.Err)
		}

		if objectStat.Key == object {
			objectMetadata.URL = *c.targetURL
			objectMetadata.Time = objectStat.LastModified
			objectMetadata.Size = objectStat.Size
			objectMetadata.Type = os.FileMode(0664)
			return objectMetadata, nil
		}

		if strings.HasSuffix(objectStat.Key, string(c.targetURL.Separator)) {
			objectMetadata.URL = *c.targetURL
			objectMetadata.Type = os.ModeDir
			return objectMetadata, nil
		}
	}
	objectStat, e := c.api.StatObject(bucket, object)
	if e != nil {
		errResponse := minio.ToErrorResponse(e)
		if errResponse.Code == "AccessDenied" {
			return nil, probe.NewError(PathInsufficientPermission{Path: c.targetURL.String()})
		}
		if errResponse.Code == "NoSuchBucket" {
			return nil, probe.NewError(BucketDoesNotExist{
				Bucket: bucket,
			})
		}
		if errResponse.Code == "InvalidBucketName" {
			return nil, probe.NewError(BucketInvalid{
				Bucket: bucket,
			})
		}
		if errResponse.Code == "NoSuchKey" || errResponse.Code == "InvalidArgument" {
			return nil, probe.NewError(ObjectMissing{})
		}
		return nil, probe.NewError(e)
	}
	objectMetadata.URL = *c.targetURL
	objectMetadata.Time = objectStat.LastModified
	objectMetadata.Size = objectStat.Size
	objectMetadata.Type = os.FileMode(0664)
	return objectMetadata, nil
}

func isAmazon(host string) bool {
	return s3utils.IsAmazonEndpoint(url.URL{Host: host})
}

func isAmazonAccelerated(host string) bool {
	return host == "s3-accelerate.amazonaws.com"
}

func isGoogle(host string) bool {
	return s3utils.IsGoogleEndpoint(url.URL{Host: host})
}

// Figure out if the URL is of 'virtual host' style.
// Currently only supported hosts with virtual style
// are Amazon S3 and Google Cloud Storage.
func isVirtualHostStyle(host string) bool {
	return isAmazon(host) || isGoogle(host) || isAmazonAccelerated(host)
}

// url2BucketAndObject gives bucketName and objectName from URL path.
func (c *s3Client) url2BucketAndObject() (bucketName, objectName string) {
	path := c.targetURL.Path
	// Convert any virtual host styled requests.
	//
	// For the time being this check is introduced for S3,
	// If you have custom virtual styled hosts please.
	// List them below.
	if c.virtualStyle {
		var bucket string
		hostIndex := strings.Index(c.targetURL.Host, "s3")
		if hostIndex == -1 {
			hostIndex = strings.Index(c.targetURL.Host, "s3-accelerate")
		}
		if hostIndex == -1 {
			hostIndex = strings.Index(c.targetURL.Host, "storage.googleapis")
		}
		if hostIndex > 0 {
			bucket = c.targetURL.Host[:hostIndex-1]
			path = string(c.targetURL.Separator) + bucket + c.targetURL.Path
		}
	}
	tokens := splitStr(path, string(c.targetURL.Separator), 3)
	return tokens[1], tokens[2]
}

// splitPath split path into bucket and object.
func (c *s3Client) splitPath(path string) (bucketName, objectName string) {
	path = strings.TrimPrefix(path, string(c.targetURL.Separator))

	// Handle path if its virtual style.
	if c.virtualStyle {
		hostIndex := strings.Index(c.targetURL.Host, "s3")
		if hostIndex == -1 {
			hostIndex = strings.Index(c.targetURL.Host, "s3-accelerate")
		}
		if hostIndex == -1 {
			hostIndex = strings.Index(c.targetURL.Host, "storage.googleapis")
		}
		if hostIndex > 0 {
			bucketName = c.targetURL.Host[:hostIndex-1]
			objectName = path
			return bucketName, objectName
		}
	}

	tokens := splitStr(path, string(c.targetURL.Separator), 2)
	return tokens[0], tokens[1]
}

/// Bucket API operations.

// List - list at delimited path, if not recursive.
func (c *s3Client) List(isRecursive, isIncomplete bool, showDir DirOpt) <-chan *clientContent {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	contentCh := make(chan *clientContent)
	if isIncomplete {
		if isRecursive {
			if showDir == DirNone {
				go c.listIncompleteRecursiveInRoutine(contentCh)
			} else {
				go c.listIncompleteRecursiveInRoutineDirOpt(contentCh, showDir)
			}
		} else {
			go c.listIncompleteInRoutine(contentCh)
		}
	} else {
		if isRecursive {
			if showDir == DirNone {
				go c.listRecursiveInRoutine(contentCh)
			} else {
				go c.listRecursiveInRoutineDirOpt(contentCh, showDir)
			}
		} else {
			go c.listInRoutine(contentCh)
		}
	}

	return contentCh
}

func (c *s3Client) listIncompleteInRoutine(contentCh chan *clientContent) {
	defer close(contentCh)
	// get bucket and object from URL.
	b, o := c.url2BucketAndObject()
	switch {
	case b == "" && o == "":
		buckets, err := c.api.ListBuckets()
		if err != nil {
			contentCh <- &clientContent{
				Err: probe.NewError(err),
			}
			return
		}
		isRecursive := false
		for _, bucket := range buckets {
			for object := range c.api.ListIncompleteUploads(bucket.Name, o, isRecursive, nil) {
				if object.Err != nil {
					contentCh <- &clientContent{
						Err: probe.NewError(object.Err),
					}
					continue
				}
				content := &clientContent{}
				url := *c.targetURL
				// Join bucket with - incoming object key.
				url.Path = c.joinPath(bucket.Name, object.Key)
				switch {
				case strings.HasSuffix(object.Key, string(c.targetURL.Separator)):
					// We need to keep the trailing Separator, do not use filepath.Join().
					content.URL = url
					content.Time = time.Now()
					content.Type = os.ModeDir
				default:
					content.URL = url
					content.Size = object.Size
					content.Time = object.Initiated
					content.Type = os.ModeTemporary
				}
				contentCh <- content
			}
		}
	default:
		isRecursive := false
		for object := range c.api.ListIncompleteUploads(b, o, isRecursive, nil) {
			if object.Err != nil {
				contentCh <- &clientContent{
					Err: probe.NewError(object.Err),
				}
				continue
			}
			content := &clientContent{}
			url := *c.targetURL
			// Join bucket with - incoming object key.
			url.Path = c.joinPath(b, object.Key)
			switch {
			case strings.HasSuffix(object.Key, string(c.targetURL.Separator)):
				// We need to keep the trailing Separator, do not use filepath.Join().
				content.URL = url
				content.Time = time.Now()
				content.Type = os.ModeDir
			default:
				content.URL = url
				content.Size = object.Size
				content.Time = object.Initiated
				content.Type = os.ModeTemporary
			}
			contentCh <- content
		}
	}
}

func (c *s3Client) listIncompleteRecursiveInRoutine(contentCh chan *clientContent) {
	defer close(contentCh)
	// get bucket and object from URL.
	b, o := c.url2BucketAndObject()
	switch {
	case b == "" && o == "":
		buckets, err := c.api.ListBuckets()
		if err != nil {
			contentCh <- &clientContent{
				Err: probe.NewError(err),
			}
			return
		}
		isRecursive := true
		for _, bucket := range buckets {
			for object := range c.api.ListIncompleteUploads(bucket.Name, o, isRecursive, nil) {
				if object.Err != nil {
					contentCh <- &clientContent{
						Err: probe.NewError(object.Err),
					}
					continue
				}
				url := *c.targetURL
				url.Path = c.joinPath(bucket.Name, object.Key)
				content := &clientContent{}
				content.URL = url
				content.Size = object.Size
				content.Time = object.Initiated
				content.Type = os.ModeTemporary
				contentCh <- content
			}
		}
	default:
		isRecursive := true
		for object := range c.api.ListIncompleteUploads(b, o, isRecursive, nil) {
			if object.Err != nil {
				contentCh <- &clientContent{
					Err: probe.NewError(object.Err),
				}
				continue
			}
			url := *c.targetURL
			// Join bucket and incoming object key.
			url.Path = c.joinPath(b, object.Key)
			content := &clientContent{}
			content.URL = url
			content.Size = object.Size
			content.Time = object.Initiated
			content.Type = os.ModeTemporary
			contentCh <- content
		}
	}
}

// Convert objectMultipartInfo to clientContent
func (c *s3Client) objectMultipartInfo2ClientContent(entry minio.ObjectMultipartInfo) clientContent {
	bucket, _ := c.url2BucketAndObject()

	content := clientContent{}
	url := *c.targetURL
	// Join bucket and incoming object key.
	url.Path = c.joinPath(bucket, entry.Key)
	content.URL = url
	content.Size = entry.Size
	content.Time = entry.Initiated

	if strings.HasSuffix(entry.Key, "/") {
		content.Type = os.ModeDir
	} else {
		content.Type = os.ModeTemporary
	}

	return content
}

// Recursively lists incomplete uploads.
func (c *s3Client) listIncompleteRecursiveInRoutineDirOpt(contentCh chan *clientContent, dirOpt DirOpt) {
	defer close(contentCh)

	// Closure function reads list of incomplete uploads and sends to contentCh. If a directory is found, it lists
	// incomplete uploads of the directory content recursively.
	var listDir func(bucket, object string) bool
	listDir = func(bucket, object string) (isStop bool) {
		isRecursive := false
		for entry := range c.api.ListIncompleteUploads(bucket, object, isRecursive, nil) {
			if entry.Err != nil {
				url := *c.targetURL
				url.Path = c.joinPath(bucket, object)
				contentCh <- &clientContent{URL: url, Err: probe.NewError(entry.Err)}

				errResponse := minio.ToErrorResponse(entry.Err)
				if errResponse.Code == "AccessDenied" {
					continue
				}

				return true
			}

			content := c.objectMultipartInfo2ClientContent(entry)

			// Handle if object.Key is a directory.
			if strings.HasSuffix(entry.Key, string(c.targetURL.Separator)) {
				if dirOpt == DirFirst {
					contentCh <- &content
				}
				if listDir(bucket, entry.Key) {
					return true
				}
				if dirOpt == DirLast {
					contentCh <- &content
				}
			} else {
				contentCh <- &content
			}
		}

		return false
	}

	bucket, object := c.url2BucketAndObject()
	listDir(bucket, object)
}

// Returns new path by joining path segments with URL path separator.
func (c *s3Client) joinPath(segments ...string) string {
	var retPath string
	pathSep := string(c.targetURL.Separator)

	for _, segment := range segments {
		segment = strings.TrimPrefix(segment, pathSep)

		if !strings.HasSuffix(retPath, pathSep) {
			retPath += pathSep
		}

		retPath += segment
	}

	return retPath
}

// Convert objectInfo to clientContent
func (c *s3Client) objectInfo2ClientContent(entry minio.ObjectInfo) clientContent {
	bucket, _ := c.url2BucketAndObject()

	content := clientContent{}
	url := *c.targetURL
	// Join bucket and incoming object key.
	url.Path = c.joinPath(bucket, entry.Key)
	content.URL = url
	content.Size = entry.Size
	content.Time = entry.LastModified

	if strings.HasSuffix(entry.Key, "/") && entry.Size == 0 && entry.LastModified.IsZero() {
		content.Type = os.ModeDir
	} else {
		content.Type = os.FileMode(0664)
	}

	return content
}

// Returns bucket stat info of current bucket.
func (c *s3Client) bucketStat() clientContent {
	bucketName, _ := c.url2BucketAndObject()

	buckets, err := c.api.ListBuckets()
	if err != nil {
		return clientContent{Err: probe.NewError(err)}
	}

	for _, bucket := range buckets {
		if bucket.Name == bucketName {
			return clientContent{URL: *c.targetURL, Time: bucket.CreationDate, Type: os.ModeDir}
		}
	}

	return clientContent{Err: probe.NewError(BucketDoesNotExist{Bucket: bucketName})}
}

// Recursively lists objects.
func (c *s3Client) listRecursiveInRoutineDirOpt(contentCh chan *clientContent, dirOpt DirOpt) {
	defer close(contentCh)

	// Closure function reads list objects and sends to contentCh. If a directory is found, it lists
	// objects of the directory content recursively.
	var listDir func(bucket, object string) bool
	listDir = func(bucket, object string) (isStop bool) {
		isRecursive := false
		for entry := range c.listObjectWrapper(bucket, object, isRecursive, nil) {
			if entry.Err != nil {
				url := *c.targetURL
				url.Path = c.joinPath(bucket, object)
				contentCh <- &clientContent{URL: url, Err: probe.NewError(entry.Err)}

				errResponse := minio.ToErrorResponse(entry.Err)
				if errResponse.Code == "AccessDenied" {
					continue
				}

				return true
			}

			content := c.objectInfo2ClientContent(entry)

			// Handle if object.Key is a directory.
			if content.Type.IsDir() {
				if dirOpt == DirFirst {
					contentCh <- &content
				}
				if listDir(bucket, entry.Key) {
					return true
				}
				if dirOpt == DirLast {
					contentCh <- &content
				}
			} else {
				contentCh <- &content
			}
		}

		return false
	}

	bucket, object := c.url2BucketAndObject()

	var cContent *clientContent

	// Get bucket stat if object is empty.
	if object == "" {
		content := c.bucketStat()
		cContent = &content

		if content.Err != nil {
			contentCh <- cContent
			return
		}
	} else if strings.HasSuffix(object, string(c.targetURL.Separator)) {
		// Get stat of given object is a directory.
		isIncomplete := false
		content, perr := c.Stat(isIncomplete)
		cContent = content
		if perr != nil {
			contentCh <- &clientContent{URL: *c.targetURL, Err: perr}
			return
		}
	}

	if cContent != nil && dirOpt == DirFirst {
		contentCh <- cContent
	}

	listDir(bucket, object)

	if cContent != nil && dirOpt == DirLast {
		contentCh <- cContent
	}
}

func (c *s3Client) listInRoutine(contentCh chan *clientContent) {
	defer close(contentCh)
	// get bucket and object from URL.
	b, o := c.url2BucketAndObject()
	switch {
	case b == "" && o == "":
		buckets, e := c.api.ListBuckets()
		if e != nil {
			contentCh <- &clientContent{
				Err: probe.NewError(e),
			}
			return
		}
		for _, bucket := range buckets {
			url := *c.targetURL
			url.Path = c.joinPath(bucket.Name)
			content := &clientContent{}
			content.URL = url
			content.Size = 0
			content.Time = bucket.CreationDate
			content.Type = os.ModeDir
			contentCh <- content
		}
	case b != "" && !strings.HasSuffix(c.targetURL.Path, string(c.targetURL.Separator)) && o == "":
		buckets, e := c.api.ListBuckets()
		if e != nil {
			contentCh <- &clientContent{
				Err: probe.NewError(e),
			}
		}
		for _, bucket := range buckets {
			if bucket.Name == b {
				content := &clientContent{}
				content.URL = *c.targetURL
				content.Size = 0
				content.Time = bucket.CreationDate
				content.Type = os.ModeDir
				contentCh <- content
				break
			}
		}
	default:
		isRecursive := false
		for object := range c.listObjectWrapper(b, o, isRecursive, nil) {
			if object.Err != nil {
				contentCh <- &clientContent{
					Err: probe.NewError(object.Err),
				}
				return
			}

			// Avoid sending an empty directory when we are specifically listing it
			if strings.HasSuffix(object.Key, string(c.targetURL.Separator)) && o == object.Key {
				continue
			}

			content := &clientContent{}
			url := *c.targetURL
			// Join bucket and incoming object key.
			url.Path = c.joinPath(b, object.Key)
			switch {
			case strings.HasSuffix(object.Key, string(c.targetURL.Separator)):
				// We need to keep the trailing Separator, do not use filepath.Join().
				content.URL = url
				content.Time = time.Now()
				content.Type = os.ModeDir
			default:
				content.URL = url
				content.Size = object.Size
				content.Time = object.LastModified
				content.Type = os.FileMode(0664)
			}
			contentCh <- content
		}
	}
}

// S3 offers a range of storage classes designed for
// different use cases, following list captures these.
const (
	// General purpose.
	// s3StorageClassStandard = "STANDARD"
	// Infrequent access.
	// s3StorageClassInfrequent = "STANDARD_IA"
	// Reduced redundancy access.
	// s3StorageClassRedundancy = "REDUCED_REDUNDANCY"
	// Archive access.
	s3StorageClassGlacier = "GLACIER"
)

func (c *s3Client) listRecursiveInRoutine(contentCh chan *clientContent) {
	defer close(contentCh)
	// get bucket and object from URL.
	b, o := c.url2BucketAndObject()
	switch {
	case b == "" && o == "":
		buckets, err := c.api.ListBuckets()
		if err != nil {
			contentCh <- &clientContent{
				Err: probe.NewError(err),
			}
			return
		}
		for _, bucket := range buckets {
			isRecursive := true
			for object := range c.listObjectWrapper(bucket.Name, o, isRecursive, nil) {
				// Return error if we encountered glacier object and continue.
				if object.StorageClass == s3StorageClassGlacier {
					contentCh <- &clientContent{
						Err: probe.NewError(ObjectOnGlacier{object.Key}),
					}
					continue
				}
				if object.Err != nil {
					contentCh <- &clientContent{
						Err: probe.NewError(object.Err),
					}
					continue
				}
				content := &clientContent{}
				objectURL := *c.targetURL
				objectURL.Path = c.joinPath(bucket.Name, object.Key)
				content.URL = objectURL
				content.Size = object.Size
				content.Time = object.LastModified
				content.Type = os.FileMode(0664)
				contentCh <- content
			}
		}
	default:
		isRecursive := true
		for object := range c.listObjectWrapper(b, o, isRecursive, nil) {
			if object.Err != nil {
				contentCh <- &clientContent{
					Err: probe.NewError(object.Err),
				}
				continue
			}
			// Ignore S3 empty directories
			if object.Size == 0 && strings.HasSuffix(object.Key, "/") {
				continue
			}
			content := &clientContent{}
			url := *c.targetURL
			// Join bucket and incoming object key.
			url.Path = c.joinPath(b, object.Key)
			content.URL = url
			content.Size = object.Size
			content.Time = object.LastModified
			content.Type = os.FileMode(0664)
			contentCh <- content
		}
	}
}

// ShareDownload - get a usable presigned object url to share.
func (c *s3Client) ShareDownload(expires time.Duration) (string, *probe.Error) {
	bucket, object := c.url2BucketAndObject()
	// No additional request parameters are set for the time being.
	reqParams := make(url.Values)
	presignedURL, e := c.api.PresignedGetObject(bucket, object, expires, reqParams)
	if e != nil {
		return "", probe.NewError(e)
	}
	return presignedURL.String(), nil
}

// ShareUpload - get data for presigned post http form upload.
func (c *s3Client) ShareUpload(isRecursive bool, expires time.Duration, contentType string) (string, map[string]string, *probe.Error) {
	bucket, object := c.url2BucketAndObject()
	p := minio.NewPostPolicy()
	if e := p.SetExpires(time.Now().UTC().Add(expires)); e != nil {
		return "", nil, probe.NewError(e)
	}
	if strings.TrimSpace(contentType) != "" || contentType != "" {
		// No need to verify for error here, since we have stripped out spaces.
		p.SetContentType(contentType)
	}
	if e := p.SetBucket(bucket); e != nil {
		return "", nil, probe.NewError(e)
	}
	if isRecursive {
		if e := p.SetKeyStartsWith(object); e != nil {
			return "", nil, probe.NewError(e)
		}
	} else {
		if e := p.SetKey(object); e != nil {
			return "", nil, probe.NewError(e)
		}
	}
	u, m, e := c.api.PresignedPostPolicy(p)
	if e != nil {
		return "", nil, probe.NewError(e)
	}
	return u.String(), m, nil
}
