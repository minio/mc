/*
 * Minio Client, (C) 2018 Minio, Inc.
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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/mimedb"
)

var (
	sqlFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "query, e",
			Usage: "sql query expression",
		},
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "sql query recursively",
		},
		cli.StringFlag{
			Name:  "icsv",
			Usage: "input csv serialization option",
		},
		cli.StringFlag{
			Name:  "ijson",
			Usage: "input json serialization option",
		},
		cli.StringFlag{
			Name:  "iparquet",
			Usage: "input parquet serialization option",
		},
		cli.StringFlag{
			Name:  "icompression",
			Usage: "input compression type",
		},
		cli.StringFlag{
			Name:  "ocsv",
			Usage: "output csv serialization option",
		},
		cli.StringFlag{
			Name:  "ojson",
			Usage: "output json serialization option",
		},
	}
)

// Display contents of a file.
var sqlCmd = cli.Command{
	Name:   "sql",
	Usage:  "run sql queries on objects",
	Action: mainSQL,
	Before: setGlobalsFromContext,
	Flags:  getSQLFlags(), //append(append(sqlFlags, ioFlags...), globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
ENVIRONMENT VARIABLES:
   MC_ENCRYPT_KEY:  list of comma delimited prefix=secret values

INPUT SERIALIZATION
	--icsv, --ijson or --iparquet flag can be accepted to describe format of object being queried.
	--icsv and --ijson accepts a string of format "key=value,..." for valid keys.

OUTPUT SERIALIZATION
	--ocsv or --ojson can be used to specify output format of data
  
COMPRESSION TYPE
	 --ocompression can specify if the queried object is compressed.
	 Valid values: NONE | GZIP | BZIP2

EXAMPLES:
   1. Run a query on a set of objects recursively on AWS S3.
      $ {{.HelpName}} --recursive --query "select * from S3Object" s3/personalbucket/my-large-csvs/

   2. Run a query on an object on Minio.
      $ {{.HelpName}} --query "select count(s.power) from S3Object" myminio/iot-devices/power-ratio.csv

   3. Run a query on an encrypted object with customer provided keys.
      $ {{.HelpName}} --encrypt-key "myminio/iot-devices=32byteslongsecretkeymustbegiven1" \
		      --query "select count(s.power) from S3Object s" myminio/iot-devices/power-ratio-encrypted.csv
	 
   4. Run a query on an object on Minio in gzip format using ; as field delimiter,
      newline as record delimiter and file header to be used 
      $ {{.HelpName}} --icompression GZIP --icsv "rd=\n,fh=USE,fd=;" \
		      --query "select count(s.power) from S3Object" myminio/iot-devices/power-ratio.csv.gz
	 
   5. Run a query on an object on Minio in gzip format using ; as field delimiter,
      newline as record delimiter and file header to be used 
      $ {{.HelpName}} --icompression GZIP --icsv "rd=\n,fh=USE,fd=;" \ 
		--ojson "rd=\n\n"	\
		--query "select count(s.power) from S3Object" myminio/iot-devices/power-ratio.csv.gz
`,
}

// filter json from allowed flags for sql command
func getSQLFlags() []cli.Flag {
	flags := append(sqlFlags, ioFlags...)
	for _, f := range globalFlags {
		if f.GetName() != "json" {
			flags = append(flags, f)
		}
	}
	return flags
}

// valid CSV and JSON keys for input/output serialization
var validCSVCommonKeys = []string{"FieldDelimiter", "QuoteChar", "QuoteEscChar"}
var validCSVInputKeys = []string{"Comments", "FileHeader", "QuotedRecordDelimiter", "RecordDelimiter"}
var validCSVOutputKeys = []string{"QuoteFields"}

var validJSONInputKeys = []string{"Type"}
var validJSONCSVCommonOutputKeys = []string{"RecordDelimiter"}

// mapping of abbreviation to long form name of CSV and JSON input/output serialization keys
var validCSVInputAbbrKeys = map[string]string{"cc": "Comments", "fh": "FileHeader", "qrd": "QuotedRecordDelimiter", "rd": "RecordDelimiter", "fd": "FieldDelimiter", "qc": "QuoteChar", "qec": "QuoteEscChar"}
var validCSVOutputAbbrKeys = map[string]string{"qf": "QuoteFields", "rd": "RecordDelimiter", "fd": "FieldDelimiter", "qc": "QuoteChar", "qec": "QuoteEscChar"}
var validJSONOutputAbbrKeys = map[string]string{"rd": "RecordDelimiter"}

// parseKVArgs parses string of the form k=v delimited by ","
// into a map of k-v pairs
func parseKVArgs(is string) (map[string]string, *probe.Error) {
	kvmap := make(map[string]string)
	var key, value string
	var s, e int  // tracking start and end of value
	var index int // current index in string
	if is != "" {
		for index < len(is) {
			i := strings.Index(is[index:], "=")
			if i == -1 {
				return nil, probe.NewError(errors.New("Arguments should be of the form key=value,... "))
			}
			key = is[index : index+i]
			s = i + index + 1
			e = strings.Index(is[s:], ",")
			delimFound := false
			for !delimFound {
				if e == -1 || e+s >= len(is) {
					delimFound = true
					break
				}
				if string(is[s+e]) != "," {
					delimFound = true
					if string(is[s+e-1]) == "," {
						e--
					}
				} else {
					e++
				}
			}
			var vEnd = len(is)
			if e != -1 {
				vEnd = s + e
			}

			value = is[s:vEnd]
			index = vEnd + 1
			if _, ok := kvmap[strings.ToLower(key)]; ok {
				return nil, probe.NewError(fmt.Errorf("More than one key=value found for %s", strings.TrimSpace(key)))
			}
			kvmap[strings.ToLower(key)] = strings.NewReplacer(`\n`, "\n", `\t`, "\t", `\r`, "\r").Replace(value)
		}
	}
	return kvmap, nil
}

// returns a string with list of serialization options and abbreviation(s) if any
func fmtString(validAbbr map[string]string) string {
	var sb strings.Builder
	i := 0
	for k, v := range validAbbr {
		sb.WriteString(fmt.Sprintf("%s(%s) ", v, k))
		i++
		if i != len(validAbbr) {
			sb.WriteString(",")
		}
	}
	return sb.String()
}

// parses the input string and constructs a k-v map, replacing any abbreviated keys with actual keys
func parseSerializationOpts(inp string, validKeys []string, validAbbrKeys map[string]string) (map[string]string, *probe.Error) {
	if validAbbrKeys == nil {
		validAbbrKeys = make(map[string]string)
	}
	validKeyFn := func(key string, validKeys []string) bool {
		for _, name := range validKeys {
			if strings.ToLower(name) == strings.ToLower(key) {
				return true
			}
		}
		return false
	}
	kv, err := parseKVArgs(inp)
	if err != nil {
		return nil, err
	}
	ikv := make(map[string]string)
	for k, v := range kv {
		fldName, ok := validAbbrKeys[strings.ToLower(k)]
		if ok {
			ikv[strings.ToLower(fldName)] = v
		} else {
			ikv[strings.ToLower(k)] = v
		}
	}
	for k := range ikv {
		if !validKeyFn(k, validKeys) {
			return nil, probe.NewError(errors.New("Options should be key-value pairs in the form key=value,... where key(s) can be one or more of " + fmtString(validAbbrKeys)))
		}
	}
	return ikv, nil
}

// gets the input serialization opts from cli context and constructs a map of csv, json or parquet options
func getInputSerializationOpts(ctx *cli.Context) map[string]map[string]string {
	icsv := ctx.String("icsv")
	ijson := ctx.String("ijson")
	iparquet := ctx.String("iparquet")
	m := make(map[string]map[string]string)

	isSet := (icsv != "")
	if ijson != "" && isSet {
		fatalIf(errInvalidArgument(), "Only one of --icsv, --ijson or --iparquet can be specified as input serialization option")
	}
	if !isSet {
		isSet = (ijson != "")
	}
	if isSet && iparquet != "" {
		fatalIf(errInvalidArgument(), "Only one of --icsv, --ijson or --iparquet can be specified as input serialization option")
	}

	if icsv != "" {
		kv, err := parseSerializationOpts(icsv, append(validCSVCommonKeys, validCSVInputKeys...), validCSVInputAbbrKeys)
		fatalIf(err, "Invalid serialization option(s) specified for --icsv flag")

		m["csv"] = kv
	}
	if ijson != "" {
		kv, err := parseSerializationOpts(ijson, validJSONInputKeys, nil)

		fatalIf(err, "Invalid serialization option(s) specified for --ijson flag")
		m["json"] = kv
	}
	if iparquet != "" {
		m["parquet"] = map[string]string{}
	}
	return m
}

// gets the output serialization opts from cli context and constructs a map of csv or json options
func getOutputSerializationOpts(ctx *cli.Context) (opts map[string]map[string]string) {
	m := make(map[string]map[string]string)
	var csvType, jsonType bool

	ocsv := ctx.String("ocsv")
	ojson := ctx.String("ojson")

	csvType = (ocsv != "")
	jsonType = (ojson != "")

	if csvType && jsonType {
		fatalIf(errInvalidArgument(), "Only one of --ocsv, or --ojson can be specified as output serialization option")
	}

	if ocsv != "" {
		validKeys := append(validCSVCommonKeys, validJSONCSVCommonOutputKeys...)
		kv, err := parseSerializationOpts(ocsv, append(validKeys, validCSVOutputKeys...), validCSVOutputAbbrKeys)
		fatalIf(err, "Invalid value(s) specified for --ocsv flag")
		m["csv"] = kv
	}
	if ojson != "" {
		kv, err := parseSerializationOpts(ojson, validJSONCSVCommonOutputKeys, validJSONOutputAbbrKeys)
		fatalIf(err, "Invalid value(s) specified for --ojson flag")
		m["json"] = kv
	}
	return m
}

func getSQLOpts(ctx *cli.Context) (s SelectObjectOpts) {
	is := getInputSerializationOpts(ctx)
	os := getOutputSerializationOpts(ctx)

	return SelectObjectOpts{
		InputSerOpts:  is,
		OutputSerOpts: os,
	}
}

func sqlSelect(targetURL, expression string, encKeyDB map[string][]prefixSSEPair, selOpts SelectObjectOpts) *probe.Error {
	alias, _, _, err := expandAlias(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}

	targetClnt, err := newClient(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}

	sseKey := getSSE(targetURL, encKeyDB[alias])
	outputer, err := targetClnt.Select(expression, sseKey, selOpts)
	if err != nil {
		return err.Trace(targetURL, expression)
	}
	defer outputer.Close()

	_, e := io.Copy(os.Stdout, outputer)
	return probe.NewError(e)
}

// check sql input arguments.
func checkSQLSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() {
		cli.ShowCommandHelpAndExit(ctx, "sql", 1) // last argument is exit code.
	}
}

// mainSQL is the main entry point for sql command.
func mainSQL(ctx *cli.Context) error {
	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

	// validate sql input arguments.
	checkSQLSyntax(ctx)
	// extract URLs.
	URLs := ctx.Args()
	query := ctx.String("query")
	selOpts := getSQLOpts(ctx)

	for _, url := range URLs {
		if !isAliasURLDir(url, encKeyDB) {
			errorIf(sqlSelect(url, query, encKeyDB, selOpts).Trace(url), "Unable to run sql")
			continue
		}
		targetAlias, targetURL, _ := mustExpandAlias(url)
		clnt, err := newClientFromAlias(targetAlias, targetURL)
		if err != nil {
			errorIf(err.Trace(url), "Unable to initialize target `"+url+"`.")
			continue
		}

		for content := range clnt.List(ctx.Bool("recursive"), false, DirNone) {
			if content.Err != nil {
				errorIf(content.Err.Trace(url), "Unable to list on target `"+url+"`.")
				continue
			}
			contentType := mimedb.TypeByExtension(filepath.Ext(content.URL.Path))
			for _, cTypeSuffix := range supportedContentTypes {
				if strings.Contains(contentType, cTypeSuffix) {
					errorIf(sqlSelect(targetAlias+content.URL.Path, query,
						encKeyDB, selOpts).Trace(content.URL.String()), "Unable to run sql")
				}
			}
		}
	}

	// Done.
	return nil
}
