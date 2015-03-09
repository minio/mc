package s3

import (
	"bytes"
	"fmt"
	"strings"

	"encoding/hex"
	"encoding/xml"
	"net/http"
)

// Error is the type returned by some API operations.
type Error struct {
	Op     string
	Code   int         // HTTP status code
	Body   []byte      // response body
	Header http.Header // response headers

	// UsedEndpoint and AmazonCode are the XML response's Endpoint and
	// Code fields, respectively.
	UseEndpoint string // if a temporary redirect (wrong endpoint)
	AmazonCode  string
}

// xmlError is the Error response from Amazon.
type xmlError struct {
	XMLName           xml.Name `xml:"Error"`
	Code              string
	Message           string
	RequestID         string
	Bucket            string
	Endpoint          string
	StringToSignBytes string
}

func (e *Error) Error() string {
	if bytes.Contains(e.Body, []byte("<Error>")) {
		return fmt.Sprintf("s3.%s: status %d: %s", e.Op, e.Code, e.Body)
	}
	return fmt.Sprintf("s3.%s: status %d", e.Op, e.Code)
}

func (e *Error) parseXML() {
	var xe xmlError
	_ = xml.NewDecoder(bytes.NewReader(e.Body)).Decode(&xe)
	e.AmazonCode = xe.Code
	if xe.Code == "TemporaryRedirect" {
		e.UseEndpoint = xe.Endpoint
	}
	if xe.Code == "SignatureDoesNotMatch" {
		want, _ := hex.DecodeString(strings.Replace(xe.StringToSignBytes, " ", "", -1))
		fmt.Printf("S3 SignatureDoesNotMatch. StringToSign should be %d bytes: %q (%x)", len(want), want, want)
	}

}
