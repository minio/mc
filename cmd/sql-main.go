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
			Name:  "csv-input",
			Usage: "csv input serialization option",
		},
		cli.StringFlag{
			Name:  "json-input",
			Usage: "json input serialization option",
		},
		cli.StringFlag{
			Name:  "compression",
			Usage: "input compression type",
		},
		cli.StringFlag{
			Name:  "csv-output",
			Usage: "csv output serialization option",
		},
		cli.StringFlag{
			Name:  "json-output",
			Usage: "json output serialization option",
		},
	}
)

// Display contents of a file.
var sqlCmd = cli.Command{
	Name:   "sql",
	Usage:  "run sql queries on objects",
	Action: mainSQL,
	Before: setGlobalsFromContext,
	Flags:  getSQLFlags(),
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
	--csv-input or --json-input can be used to specify input data format. Format is specified by a string
	with pattern "key=value,..." for valid key(s).
	DATA FORMAT:
	 csv: Use --csv-input flag
				Valid keys:
				RecordDelimiter (rd)
				FieldDelimiter (fd)
				QuoteChar (qc)
				QuoteEscChar (qec)
				FileHeader (fh)
				Comments (cc)
				QuotedRecordDelimiter (qrd)
	 
   json: Use --json-input flag
				Valid keys:
				Type 
	 parquet: If object name ends in .parquet, this is automatically interpreted.
	  
OUTPUT SERIALIZATION
	--csv-output or --json-output can be used to specify output data format. Format is specified by a string
	with pattern "key=value,..." for valid key(s).
	DATA FORMAT:
	 csv: Use --csv-output flag
				Valid keys:
				RecordDelimiter (rd)
				FieldDelimiter (fd)
				QuoteChar (qc)
				QuoteEscChar (qec)
				QuoteFields (qf)
	 
    json: Use --json-output flag
				Valid keys:
				RecordDelimiter (rd) 
	  
COMPRESSION TYPE
	 --compression specifies if the queried object is compressed.
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
      $ {{.HelpName}} --compression GZIP --csv-input "rd=\n,fh=USE,fd=;" \
		      --query "select count(s.power) from S3Object" myminio/iot-devices/power-ratio.csv.gz
	 
   5. Run a query on an object on Minio in gzip format using ; as field delimiter,
      newline as record delimiter and file header to be used 
      $ {{.HelpName}} --compression GZIP --csv-input "rd=\n,fh=USE,fd=;" \ 
               --json-output "rd=\n\n" --query "select * from S3Object" myminio/iot-devices/data.csv

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
func fmtString(validAbbr map[string]string, validKeys []string) string {
	var sb strings.Builder
	i := 0
	for k, v := range validAbbr {
		sb.WriteString(fmt.Sprintf("%s(%s) ", v, k))
		i++
		if i != len(validAbbr) {
			sb.WriteString(",")
		}
	}
	if len(sb.String()) == 0 {
		for _, k := range validKeys {
			sb.WriteString(fmt.Sprintf("%s ", k))
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
			return nil, probe.NewError(errors.New("Options should be key-value pairs in the form key=value,... where valid key(s) are " + fmtString(validAbbrKeys, validKeys)))
		}
	}
	return ikv, nil
}

// gets the input serialization opts from cli context and constructs a map of csv, json or parquet options
func getInputSerializationOpts(ctx *cli.Context) map[string]map[string]string {
	icsv := ctx.String("csv-input")
	ijson := ctx.String("json-input")
	// iparquet := ctx.String("iparquet")
	m := make(map[string]map[string]string)

	csvType := (icsv != "")
	jsonType := (ijson != "")
	if csvType && jsonType {
		fatalIf(errInvalidArgument(), "Only one of --csv-input or --json-input can be specified as input serialization option")
	}

	if icsv != "" {
		kv, err := parseSerializationOpts(icsv, append(validCSVCommonKeys, validCSVInputKeys...), validCSVInputAbbrKeys)
		fatalIf(err, "Invalid serialization option(s) specified for --csv-input flag")

		m["csv"] = kv
	}
	if ijson != "" {
		kv, err := parseSerializationOpts(ijson, validJSONInputKeys, nil)

		fatalIf(err, "Invalid serialization option(s) specified for --json-input flag")
		m["json"] = kv
	}
	// if iparquet != "" {
	// 	m["parquet"] = map[string]string{}
	// }
	return m
}

// gets the output serialization opts from cli context and constructs a map of csv or json options
func getOutputSerializationOpts(ctx *cli.Context) (opts map[string]map[string]string) {
	m := make(map[string]map[string]string)
	var csvType, jsonType bool

	ocsv := ctx.String("csv-output")
	ojson := ctx.String("json-output")
	csvType = (ocsv != "")
	jsonType = (ojson != "")

	if csvType && jsonType {
		fatalIf(errInvalidArgument(), "Only one of --csv-output, or --json-output can be specified as output serialization option")
	}

	if ocsv != "" {
		validKeys := append(validCSVCommonKeys, validJSONCSVCommonOutputKeys...)
		kv, err := parseSerializationOpts(ocsv, append(validKeys, validCSVOutputKeys...), validCSVOutputAbbrKeys)
		fatalIf(err, "Invalid value(s) specified for --csv-output flag")
		m["csv"] = kv
	}
	// default to JSON format if no output option is specified
	if ojson == "" && ocsv == "" {
		ojson = "rd=\n"
	}

	if ojson != "" {
		kv, err := parseSerializationOpts(ojson, validJSONCSVCommonOutputKeys, validJSONOutputAbbrKeys)
		fatalIf(err, "Invalid value(s) specified for --json-output flag")
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
func validateOpts(selOpts SelectObjectOpts, url string) {
	_, targetURL, _ := mustExpandAlias(url)
	if strings.HasSuffix(targetURL, ".parquet") && (selOpts.InputSerOpts != nil || selOpts.OutputSerOpts != nil) {
		fatalIf(errInvalidArgument(), "Input serialization flags --csv-input and --json-input cannot be used for object in .parquet format")
	}
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
			validateOpts(selOpts, url)
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
