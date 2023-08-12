package cmd

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/minio/mc/cmd/utils"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/minio/minio-go/v7/pkg/signer"
	"golang.org/x/net/publicsuffix"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"net/url"
	"runtime"
	"strings"
	"sync/atomic"
)

type ZcnClient struct {
	// Parsed endpoint url provided by the user.
	endpointURL  *ClientURL
	httpClient   *http.Client
	healthStatus int32
	// Holds various credential providers.
	credsProvider *credentials.Credentials

	// Custom signerType value overrides all credentials.
	overrideSignerType credentials.SignatureType

	// User supplied.
	appInfo struct {
		appName    string
		appVersion string
	}
}

const (
	unknown = -1
	offline = 0
	online  = 1
)

// Global constants.
const (
	libraryName    = "minio-go"
	libraryVersion = "v7.0.62"
	amzVersionID   = "X-Amz-Version-Id"
)
const libraryUserAgentPrefix = "MinIO (" + runtime.GOOS + "; " + runtime.GOARCH + ") "
const libraryUserAgent = libraryUserAgentPrefix + libraryName + "/" + libraryVersion

// maxMultipartPutObjectSize - maximum size 5TiB of object for
// Multipart operation.
const maxMultipartPutObjectSize = 1024 * 1024 * 1024 * 1024 * 5

// unsignedPayload - value to be set to X-Amz-Content-Sha256 header when
// we don't want to sign the request payload
const unsignedPayload = "UNSIGNED-PAYLOAD"

// unsignedPayloadTrailer value to be set to X-Amz-Content-Sha256 header when
// we don't want to sign the request payload, but have a trailer.
const unsignedPayloadTrailer = "STREAMING-UNSIGNED-PAYLOAD-TRAILER"

func NewZcnClient(config *Config, tripper http.RoundTripper, targetUrl *ClientURL) *ZcnClient {
	// Initialize cookies to preserve server sent cookies if any and replay
	// them upon each request.
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil
	}
	return &ZcnClient{
		endpointURL: targetUrl,
		httpClient: &http.Client{
			Jar:       jar,
			Transport: tripper,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}},
		healthStatus:       unknown,
		credsProvider:      credentials.NewStaticV4(config.AccessKey, config.SecretKey, config.SessionToken),
		overrideSignerType: credentials.SignatureV4,
		appInfo: struct {
			appName    string
			appVersion string
		}{appName: config.AppName, appVersion: config.AppVersion},
	}
}

// errEntityTooLarge - Input size is larger than supported maximum.
func errEntityTooLarge(totalSize, maxObjectSize int64, bucketName, objectName string) error {
	msg := fmt.Sprintf("Your proposed upload size ‘%d’ exceeds the maximum allowed object size ‘%d’ for single PUT operation.", totalSize, maxObjectSize)
	return minio.ErrorResponse{
		StatusCode: http.StatusBadRequest,
		Code:       "EntityTooLarge",
		Message:    msg,
		BucketName: bucketName,
		Key:        objectName,
	}
}

// errEntityTooSmall - Input size is smaller than supported minimum.
func errEntityTooSmall(totalSize int64, bucketName, objectName string) error {
	msg := fmt.Sprintf("Your proposed upload size ‘%d’ is below the minimum allowed object size ‘0B’ for single PUT operation.", totalSize)
	return minio.ErrorResponse{
		StatusCode: http.StatusBadRequest,
		Code:       "EntityTooSmall",
		Message:    msg,
		BucketName: bucketName,
		Key:        objectName,
	}
}

func (c *ZcnClient) PutMultipleObjects(
	ctx context.Context,
	bucketName string,
	objectNames []string,
	readers []io.Reader,
	objectSizes []int64,
	opts minio.PutObjectOptions,
) (info minio.UploadInfo, err error) {
	n := len(objectSizes)
	for i := 0; i < n; i++ {
		if objectSizes[i] < 0 && opts.DisableMultipart {
			return minio.UploadInfo{}, errors.New("object size must be provided with disable multipart upload")
		}
	}

	for i := 0; i < n; i++ {
		if objectSizes[i] > int64(maxMultipartPutObjectSize) {
			return minio.UploadInfo{}, errEntityTooLarge(objectSizes[i], maxMultipartPutObjectSize, bucketName, "")
		}

		if objectSizes[i] <= 0 {
			return minio.UploadInfo{}, errEntityTooSmall(objectSizes[i], bucketName, "")
		}
	}

	return c.putMultipleObject(ctx, bucketName, objectNames, readers, objectSizes, opts)
}

// putMultipleObject puts multiple object into minio s3 compatible server.
func (c *ZcnClient) putMultipleObject(
	ctx context.Context,
	bucketName string,
	objectNames []string,
	readers []io.Reader,
	sizes []int64,
	opts minio.PutObjectOptions,
) (info minio.UploadInfo, err error) {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return minio.UploadInfo{}, err
	}

	// Create a new multipart writer
	var bodyBuf bytes.Buffer
	bodyWriter := multipart.NewWriter(&bodyBuf)

	// Loop through the readers and add them as parts to the multipart form
	for i, reader := range readers {
		fileName := objectNames[i]
		fileSize := sizes[i]

		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fileName, fileName))

		partWriter, err := bodyWriter.CreatePart(h)
		if err != nil {
			return minio.UploadInfo{}, err
		}

		_, err = io.CopyN(partWriter, reader, fileSize)
		if err != nil {
			return minio.UploadInfo{}, err
		}
	}

	// Close the multipart writer to finalize the body
	bodyWriter.Close()

	// Create the request
	urlValues := make(url.Values)
	urlValues.Set("multiupload", "true")
	urlStr, _ := c.makeTargetURL(bucketName, "", "", false, urlValues)

	hook := utils.NewHook(&bodyBuf, opts.Progress)
	req, err := http.NewRequest(http.MethodPut, urlStr.String(), hook)
	if err != nil {
		return minio.UploadInfo{}, err
	}

	// Set the required headers for multipart form data
	req.Header.Set("Content-Type", bodyWriter.FormDataContentType())

	// Get credentials from the configured credentials provider.
	value, err := c.credsProvider.Get()
	if err != nil {
	}

	var (
		signerType      = value.SignerType
		accessKeyID     = value.AccessKeyID
		secretAccessKey = value.SecretAccessKey
		sessionToken    = value.SessionToken
	)

	// Custom signer set then override the behavior.
	if c.overrideSignerType != credentials.SignatureDefault {
		signerType = c.overrideSignerType
	}

	// If signerType returned by credentials helper is anonymous,
	// then do not sign regardless of signerType override.
	if value.SignerType == credentials.SignatureAnonymous {
		signerType = credentials.SignatureAnonymous
	}

	c.setUserAgent(req)

	// Set all headers.
	for k, v := range opts.UserMetadata {
		req.Header.Set(k, v)
	}

	switch {
	case signerType.IsV2():
		// Add signature version '2' authorization header.
		req = signer.SignV2(*req, accessKeyID, secretAccessKey, false)
	default:
		// Set sha256 sum for signature calculation only with signature version '4'.
		shaHeader := unsignedPayload
		shaHeader = unsignedPayloadTrailer
		req.Header.Set("X-Amz-Content-Sha256", shaHeader)

		// Add signature version '4' authorization header.
		req = signer.SignV4Trailer(*req, accessKeyID, secretAccessKey, sessionToken, "", make(http.Header, 1))
	}

	// Execute the request
	resp, err := c.do(ctx, req)
	if err != nil {
		return minio.UploadInfo{}, err
	}

	defer closeResponse(resp)
	if err != nil {
		return minio.UploadInfo{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			fmt.Println("failure error response with status code: ", resp.StatusCode)
			return minio.UploadInfo{}, httpRespToErrorResponse(resp, bucketName, "")
		}
	}

	// extract lifecycle expiry date and rule ID
	h := resp.Header
	return minio.UploadInfo{
		Bucket:    bucketName,
		Key:       "",
		ETag:      trimEtag(h.Get("ETag")),
		VersionID: h.Get(amzVersionID),
		Size:      sizes[0],

		// Checksum values
		ChecksumCRC32:  h.Get("x-amz-checksum-crc32"),
		ChecksumCRC32C: h.Get("x-amz-checksum-crc32c"),
		ChecksumSHA1:   h.Get("x-amz-checksum-sha1"),
		ChecksumSHA256: h.Get("x-amz-checksum-sha256"),
	}, nil

}

func (c *ZcnClient) do(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	if c.IsOffline() {
		return nil, errors.New(c.endpointURL.String() + " is offline.")
	}

	_, cancel := context.WithCancel(ctx)

	defer cancel()
	resp, err = c.httpClient.Do(req)
	if err != nil {
		// Handle this specifically for now until future Golang versions fix this issue properly.
		if urlErr, ok := err.(*url.Error); ok {
			if strings.Contains(urlErr.Err.Error(), "EOF") {
				return nil, &url.Error{
					Op:  urlErr.Op,
					URL: urlErr.URL,
					Err: errors.New("Connection closed by foreign host " + urlErr.URL + ". Retry again."),
				}
			}
		}
		return nil, err
	}

	// Response cannot be non-nil, report error if thats the case.
	if resp == nil {
		msg := "Response is empty. " + reportIssue
		return nil, minio.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Code:       "InvalidArgument",
			Message:    msg,
			RequestID:  "minio",
		}
	}

	return resp, nil
}

func (c *ZcnClient) markOffline() {
	atomic.CompareAndSwapInt32(&c.healthStatus, online, offline)
}

// makeTargetURL make a new target url.
func (c *ZcnClient) makeTargetURL(bucketName, objectName, bucketLocation string, isVirtualHostStyle bool, queryValues url.Values) (*url.URL, error) {
	host := c.endpointURL.Host

	// Save scheme.
	scheme := c.endpointURL.Scheme

	// Strip port 80 and 443 so we won't send these ports in Host header.
	// The reason is that browsers and curl automatically remove :80 and :443
	// with the generated presigned urls, then a signature mismatch error.
	if h, p, err := net.SplitHostPort(host); err == nil {
		if scheme == "http" && p == "80" || scheme == "https" && p == "443" {
			host = h
			if ip := net.ParseIP(h); ip != nil && ip.To4() == nil {
				host = "[" + h + "]"
			}
		}
	}

	urlStr := scheme + "://" + host + "/"

	// Make URL only if bucketName is available, otherwise use the
	// endpoint URL.
	if bucketName != "" {
		// If endpoint supports virtual host style use that always.
		// Currently only S3 and Google Cloud Storage would support
		// virtual host style.
		if isVirtualHostStyle {
			urlStr = scheme + "://" + bucketName + "." + host + "/"
			if objectName != "" {
				urlStr += s3utils.EncodePath(objectName)
			}
		} else {
			// If not fall back to using path style.
			urlStr = urlStr + bucketName + "/"
			if objectName != "" {
				urlStr += s3utils.EncodePath(objectName)
			}
		}
	}

	// If there are any query values, add them to the end.
	if len(queryValues) > 0 {
		urlStr = urlStr + "?" + s3utils.QueryEncode(queryValues)
	}

	return url.Parse(urlStr)
}

// set User agent.
func (c *ZcnClient) setUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", libraryUserAgent)
	if c.appInfo.appName != "" && c.appInfo.appVersion != "" {
		req.Header.Set("User-Agent", libraryUserAgent+" "+c.appInfo.appName+"/"+c.appInfo.appVersion)
	}
}

func closeResponse(resp *http.Response) {
	// Callers should close resp.Body when done reading from it.
	// If resp.Body is not closed, the Client's underlying RoundTripper
	// (typically Transport) may not be able to re-use a persistent TCP
	// connection to the server for a subsequent "keep-alive" request.
	if resp != nil && resp.Body != nil {
		// Drain any remaining Body and then close the connection.
		// Without this closing connection would disallow re-using
		// the same connection for future uses.
		//  - http://stackoverflow.com/a/17961593/4465767
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

const (
	reportIssue = "Please report this issue at https://github.com/minio/minio-go/issues."
)

// httpRespToErrorResponse returns a new encoded ErrorResponse
// structure as error.
func httpRespToErrorResponse(resp *http.Response, bucketName, objectName string) error {
	if resp == nil {
		msg := "Empty http response. " + reportIssue
		return minio.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Code:       "InvalidArgument",
			Message:    msg,
			RequestID:  "minio",
		}
	}

	errResp := minio.ErrorResponse{
		StatusCode: resp.StatusCode,
		Server:     resp.Header.Get("Server"),
	}

	errBody, err := xmlDecodeAndBody(resp.Body, &errResp)
	// Xml decoding failed with no body, fall back to HTTP headers.
	if err != nil {
		switch resp.StatusCode {
		case http.StatusNotFound:
			if objectName == "" {
				errResp = minio.ErrorResponse{
					StatusCode: resp.StatusCode,
					Code:       "NoSuchBucket",
					Message:    "The specified bucket does not exist.",
					BucketName: bucketName,
				}
			} else {
				errResp = minio.ErrorResponse{
					StatusCode: resp.StatusCode,
					Code:       "NoSuchKey",
					Message:    "The specified key does not exist.",
					BucketName: bucketName,
					Key:        objectName,
				}
			}
		case http.StatusForbidden:
			errResp = minio.ErrorResponse{
				StatusCode: resp.StatusCode,
				Code:       "AccessDenied",
				Message:    "Access Denied.",
				BucketName: bucketName,
				Key:        objectName,
			}
		case http.StatusConflict:
			errResp = minio.ErrorResponse{
				StatusCode: resp.StatusCode,
				Code:       "Conflict",
				Message:    "Bucket not empty.",
				BucketName: bucketName,
			}
		case http.StatusPreconditionFailed:
			errResp = minio.ErrorResponse{
				StatusCode: resp.StatusCode,
				Code:       "PreconditionFailed",
				Message:    s3ErrorResponseMap["PreconditionFailed"],
				BucketName: bucketName,
				Key:        objectName,
			}
		default:
			msg := resp.Status
			if len(errBody) > 0 {
				msg = string(errBody)
				if len(msg) > 1024 {
					msg = msg[:1024] + "..."
				}
			}
			errResp = minio.ErrorResponse{
				StatusCode: resp.StatusCode,
				Code:       resp.Status,
				Message:    msg,
				BucketName: bucketName,
			}
		}
	}

	code := resp.Header.Get("x-minio-error-code")
	if code != "" {
		errResp.Code = code
	}
	desc := resp.Header.Get("x-minio-error-desc")
	if desc != "" {
		errResp.Message = strings.Trim(desc, `"`)
	}

	// Save hostID, requestID and region information
	// from headers if not available through error XML.
	if errResp.RequestID == "" {
		errResp.RequestID = resp.Header.Get("x-amz-request-id")
	}
	if errResp.HostID == "" {
		errResp.HostID = resp.Header.Get("x-amz-id-2")
	}
	if errResp.Region == "" {
		errResp.Region = resp.Header.Get("x-amz-bucket-region")
	}
	if errResp.Code == "InvalidRegion" && errResp.Region != "" {
		errResp.Message = fmt.Sprintf("Region does not match, expecting region ‘%s’.", errResp.Region)
	}

	return errResp
}

// xmlDecodeAndBody reads the whole body up to 1MB and
// tries to XML decode it into v.
// The body that was read and any error from reading or decoding is returned.
func xmlDecodeAndBody(bodyReader io.Reader, v interface{}) ([]byte, error) {
	// read the whole body (up to 1MB)
	const maxBodyLength = 1 << 20
	body, err := io.ReadAll(io.LimitReader(bodyReader, maxBodyLength))
	fmt.Println("body: ", string(body))
	fmt.Println("error: ", err.Error())
	if err != nil {
		return nil, err
	}
	return bytes.TrimSpace(body), xmlDecoder(bytes.NewReader(body), v)
}

func xmlDecoder(body io.Reader, v interface{}) error {
	d := xml.NewDecoder(body)
	return d.Decode(v)
}

func trimEtag(etag string) string {
	etag = strings.TrimPrefix(etag, "\"")
	return strings.TrimSuffix(etag, "\"")
}

// Non-exhaustive list of AWS S3 standard error responses -
// http://docs.aws.amazon.com/AmazonS3/latest/API/ErrorResponses.html
var s3ErrorResponseMap = map[string]string{
	"AccessDenied":                      "Access Denied.",
	"BadDigest":                         "The Content-Md5 you specified did not match what we received.",
	"EntityTooSmall":                    "Your proposed upload is smaller than the minimum allowed object size.",
	"EntityTooLarge":                    "Your proposed upload exceeds the maximum allowed object size.",
	"IncompleteBody":                    "You did not provide the number of bytes specified by the Content-Length HTTP header.",
	"InternalError":                     "We encountered an internal error, please try again.",
	"InvalidAccessKeyId":                "The access key ID you provided does not exist in our records.",
	"InvalidBucketName":                 "The specified bucket is not valid.",
	"InvalidDigest":                     "The Content-Md5 you specified is not valid.",
	"InvalidRange":                      "The requested range is not satisfiable",
	"MalformedXML":                      "The XML you provided was not well-formed or did not validate against our published schema.",
	"MissingContentLength":              "You must provide the Content-Length HTTP header.",
	"MissingContentMD5":                 "Missing required header for this request: Content-Md5.",
	"MissingRequestBodyError":           "Request body is empty.",
	"NoSuchBucket":                      "The specified bucket does not exist.",
	"NoSuchBucketPolicy":                "The bucket policy does not exist",
	"NoSuchKey":                         "The specified key does not exist.",
	"NoSuchUpload":                      "The specified multipart upload does not exist. The upload ID may be invalid, or the upload may have been aborted or completed.",
	"NotImplemented":                    "A header you provided implies functionality that is not implemented",
	"PreconditionFailed":                "At least one of the pre-conditions you specified did not hold",
	"RequestTimeTooSkewed":              "The difference between the request time and the server's time is too large.",
	"SignatureDoesNotMatch":             "The request signature we calculated does not match the signature you provided. Check your key and signing method.",
	"MethodNotAllowed":                  "The specified method is not allowed against this resource.",
	"InvalidPart":                       "One or more of the specified parts could not be found.",
	"InvalidPartOrder":                  "The list of parts was not in ascending order. The parts list must be specified in order by part number.",
	"InvalidObjectState":                "The operation is not valid for the current state of the object.",
	"AuthorizationHeaderMalformed":      "The authorization header is malformed; the region is wrong.",
	"MalformedPOSTRequest":              "The body of your POST request is not well-formed multipart/form-data.",
	"BucketNotEmpty":                    "The bucket you tried to delete is not empty",
	"AllAccessDisabled":                 "All access to this bucket has been disabled.",
	"MalformedPolicy":                   "Policy has invalid resource.",
	"MissingFields":                     "Missing fields in request.",
	"AuthorizationQueryParametersError": "Error parsing the X-Amz-Credential parameter; the Credential is mal-formed; expecting \"<YOUR-AKID>/YYYYMMDD/REGION/SERVICE/aws4_request\".",
	"MalformedDate":                     "Invalid date format header, expected to be in ISO8601, RFC1123 or RFC1123Z time format.",
	"BucketAlreadyOwnedByYou":           "Your previous request to create the named bucket succeeded and you already own it.",
	"InvalidDuration":                   "Duration provided in the request is invalid.",
	"XAmzContentSHA256Mismatch":         "The provided 'x-amz-content-sha256' header does not match what was computed.",
	// Add new API errors here.
}

// IsOffline returns true if healthcheck enabled and client is offline
// If HealthCheck function has not been called this will always return false.
func (c *ZcnClient) IsOffline() bool {
	return atomic.LoadInt32(&c.healthStatus) == offline
}
