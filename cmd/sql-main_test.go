/*
 * MinIO Client (C) 2019 MinIO, Inc.
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
	"strings"
	"testing"
)

var testParseKVArgsCases = []struct {
	inp    string
	kvmap  map[string]string
	errMsg string
}{
	{"fh=use,rd=|,fd=;,qec=\"", map[string]string{"fh": "use", "rd": "|", "fd": ";", "qec": "\""}, "<nil>"},
	{"", map[string]string{}, "<nil>"},
	{"not the right format", map[string]string{}, "Arguments should be of the form key=value,... "},
	{"k==v", map[string]string{"k": "=v"}, "<nil>"},
	{"k=v1,k=v2", map[string]string{}, "More than one key=value found for k"},
	{"k=v1;k=v2", map[string]string{"k": "v1;k=v2"}, "<nil>"},
}

func TestParseKVArgs(t *testing.T) {
	for _, test := range testParseKVArgsCases {
		kvmap, err := parseKVArgs(test.inp)
		gerr := err.ToGoError()
		if gerr != nil && gerr.Error() != test.errMsg {
			t.Fatalf("Unexpected result for \"%s\", expected: |%s|  got: |%s|\n", test.inp, test.errMsg, gerr)
		}
		if gerr == nil && test.errMsg != "<nil>" {
			t.Fatalf("Unexpected result for \"%s\", expected: |%s|  got: |%s|\n", test.inp, test.errMsg, gerr)
		}
		for k, v := range test.kvmap {
			actual, ok := kvmap[k]
			if !ok {
				t.Fatalf("Unexpected result for \"%s,\" expected %s , found %s for key %s\n", test.inp, v, actual, k)
			}
		}
	}
}

var testParseSerializationCases = []struct {
	inp           string
	validKeys     []string
	validAbbrKeys map[string]string
	parsedOpts    map[string]string
	errMsg        string
}{
	{"rd=\n,fd=;,qc=\"",
		append(validCSVCommonKeys, validCSVInputKeys...),
		validCSVInputAbbrKeys,
		map[string]string{"recorddelimiter": "\n", "fielddelimiter": ";", "quotechar": "\""},
		"<nil>",
	},
	{"rd=\n,fd=;,qc=\"",
		validCSVInputKeys,
		validCSVInputAbbrKeys,
		map[string]string{},
		"Options should be key-value pairs in the form key=value,... where valid key(s) are ",
	},
	{"nokey=\n,fd=;,qc=\"",
		validCSVInputKeys,
		validCSVInputAbbrKeys,
		map[string]string{},
		"Options should be key-value pairs in the form key=value,... where valid key(s) are ",
	},
	{"rd=\n\n,fd=|,qc=\",qc='",
		validCSVInputKeys,
		validCSVInputAbbrKeys,
		map[string]string{},
		"More than one key=value found for ",
	},
	{"recordDelimiter=\n\n,FieldDelimiter=|,QuoteChAR=\"",
		append(validCSVCommonKeys, validCSVInputKeys...),
		validCSVInputAbbrKeys,
		map[string]string{"recorddelimiter": "\n\n", "fielddelimiter": "|", "quotechar": "\""},
		"<nil>",
	},
	{"recordDelimiter=\n\n,FieldDelimiter=|,QuoteChAR=\",fh=use,qrd=;",
		append(validCSVCommonKeys, validCSVInputKeys...),
		validCSVInputAbbrKeys,
		map[string]string{"recorddelimiter": "\n\n", "fielddelimiter": "|", "quotechar": "\"", "quotedrecorddelimiter": ";", "fileheader": "use"},
		"<nil>",
	},
	{"recordDelimiter=\n\n,FieldDelimiter=|,QuoteChar=\",qf=;,qec='",
		append(validCSVCommonKeys, validCSVOutputKeys...),
		validCSVOutputAbbrKeys,
		map[string]string{},
		"Options should be key-value pairs in the form key=value,... where valid key(s) are ",
	},
	{"FieldDelimiter=|,QuoteChar=\",qf=;,qec='",
		append(validCSVCommonKeys, validCSVOutputKeys...),
		validCSVOutputAbbrKeys,
		map[string]string{"fielddelimiter": "|", "quotechar": "\"", "quotefields": ";", "quoteescchar": "'"},
		"<nil>",
	},
	{"type=lines",
		validJSONInputKeys,
		nil,
		map[string]string{"type": "lines"},
		"<nil>",
	},
}

func TestParseSerializationOpts(t *testing.T) {

	for i, test := range testParseSerializationCases {
		optsMap, err := parseSerializationOpts(test.inp, test.validKeys, test.validAbbrKeys)
		gerr := err.ToGoError()
		if gerr != nil && gerr.Error() != test.errMsg {
			// match partial error message
			if !strings.Contains(gerr.Error(), test.errMsg) {
				t.Fatalf("Test %d: Unexpected result for \"%s\", expected: |%s|  got: |%s|\n", i+1, test.inp, test.errMsg, gerr)
			}
		}
		if gerr == nil && test.errMsg != "<nil>" {
			t.Fatalf("Test %d: Unexpected result for \"%s\", expected: |%s|  got: |%s|\n", i+1, test.inp, test.errMsg, gerr)
		}
		for k, v := range test.parsedOpts {
			actual, ok := optsMap[strings.ToLower(k)]
			if !ok {
				t.Fatalf("Test %d:Unexpected result for \"%s,\" expected %s , found %s for key %s\n", i+1, test.inp, v, actual, k)
			}
		}
	}
}
