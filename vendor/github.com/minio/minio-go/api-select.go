/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage
 * (C) 2018 Minio, Inc.
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
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// CSVFileHeaderInfo -
type CSVFileHeaderInfo string

// Constants for file header info.
const (
	CSVFileHeaderInfoNone   CSVFileHeaderInfo = "NONE"
	CSVFileHeaderInfoIgnore                   = "IGNORE"
	CSVFileHeaderInfoUse                      = "USE"
)

// SelectCompressionType -
type SelectCompressionType string

// Constants for compression types under select API.
const (
	SelectCompressionNONE SelectCompressionType = "NONE"
	SelectCompressionGZIP                       = "GZIP"
)

// CSVQuoteFields -
type CSVQuoteFields string

// Constants for csv quote styles.
const (
	CSVQuoteFieldsAlways   CSVQuoteFields = "Always"
	CSVQuoteFieldsAsNeeded                = "AsNeeded"
)

// QueryExpressionType -
type QueryExpressionType string

// Constants for expression type.
const (
	QueryExpressionTypeSQL QueryExpressionType = "SQL"
)

// JSONType determines json input serialization type.
type JSONType string

// Constants for JSONTypes.
const (
	JSONDocumentType JSONType = "Document"
	JSONStreamType            = "Stream"
	JSONLinesType             = "Lines"
)

// ObjectSelectRequest - represents the input select body
type ObjectSelectRequest struct {
	XMLName            xml.Name `xml:"SelectRequest" json:"-"`
	Expression         string
	ExpressionType     QueryExpressionType
	InputSerialization struct {
		CompressionType SelectCompressionType
		CSV             struct {
			FileHeaderInfo       CSVFileHeaderInfo
			RecordDelimiter      string
			FieldDelimiter       string
			QuoteCharacter       string
			QuoteEscapeCharacter string
			Comments             string
		}
		JSON struct {
			Type JSONType
		}
	}
	OutputSerialization struct {
		CSV struct {
			QuoteFields          CSVQuoteFields
			RecordDelimiter      string
			FieldDelimiter       string
			QuoteCharacter       string
			QuoteEscapeCharacter string
		}
		JSON struct {
			RecordDelimiter string
		}
	}
}

// SelectObjectType -
type SelectObjectType string

// Constants for input data types.
const (
	SelectObjectTypeCSV  SelectObjectType = "CSV"
	SelectObjectTypeJSON                  = "JSON"
)

// SelectObjectOptions -
type SelectObjectOptions struct {
	Type  SelectObjectType
	Input struct {
		RecordDelimiter string
		FieldDelimiter  string
		Comments        string
	}
	Output struct {
		RecordDelimiter string
		FieldDelimiter  string
	}
}

// ToObjectSelectRequest - generate an object select request statement.
func (opts SelectObjectOptions) ToObjectSelectRequest(expression string, compressionType SelectCompressionType) *ObjectSelectRequest {
	osreq := &ObjectSelectRequest{
		Expression:     expression,
		ExpressionType: QueryExpressionTypeSQL,
	}
	osreq.InputSerialization.CompressionType = compressionType
	if opts.Type == SelectObjectTypeCSV {
		osreq.InputSerialization.CSV.FieldDelimiter = opts.Input.FieldDelimiter
		osreq.InputSerialization.CSV.RecordDelimiter = opts.Input.RecordDelimiter
		osreq.InputSerialization.CSV.Comments = opts.Input.Comments
		// Only supports filed and record delimiter for now.
		osreq.OutputSerialization.CSV.FieldDelimiter = opts.Output.FieldDelimiter
		osreq.OutputSerialization.CSV.RecordDelimiter = opts.Output.RecordDelimiter
	} else if opts.Type == SelectObjectTypeJSON {
		// FIXME: Only supporting JSON document type for now.
		osreq.InputSerialization.JSON.Type = JSONDocumentType
		osreq.OutputSerialization.JSON.RecordDelimiter = opts.Output.RecordDelimiter
	}
	return osreq
}

// SelectObjectContent is a implementation of http://docs.aws.amazon.com/AmazonS3/latest/API/RESTObjectSELECTContent.html AWS S3 API.
func (c Client) SelectObjectContent(ctx context.Context, bucketName, objectName, expression string, opts SelectObjectOptions) (io.ReadCloser, error) {
	objInfo, err := c.statObject(ctx, bucketName, objectName, StatObjectOptions{})
	if err != nil {
		return nil, err
	}

	var compressionType = SelectCompressionNONE
	if strings.Contains(objInfo.ContentType, "gzip") {
		compressionType = SelectCompressionGZIP
	} else if strings.Contains(objInfo.ContentType, "text/csv") && opts.Type == "" {
		opts.Type = SelectObjectTypeCSV
	} else if strings.Contains(objInfo.ContentType, "json") && opts.Type == "" {
		opts.Type = SelectObjectTypeJSON
	}

	selectReqBytes, err := xml.Marshal(opts.ToObjectSelectRequest(expression, compressionType))
	if err != nil {
		return nil, err
	}

	urlValues := make(url.Values)
	urlValues.Set("select", "")
	urlValues.Set("select-type", "2")

	// Execute POST on bucket/object.
	resp, err := c.executeMethod(ctx, "POST", requestMetadata{
		bucketName:       bucketName,
		objectName:       objectName,
		queryValues:      urlValues,
		contentMD5Base64: sumMD5Base64(selectReqBytes),
		contentSHA256Hex: sum256Hex(selectReqBytes),
		contentBody:      bytes.NewReader(selectReqBytes),
		contentLength:    int64(len(selectReqBytes)),
	})
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp, bucketName, "")
	}

	return resp.Body, nil
}
