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
	// "bytes"

	"encoding/xml"
	"fmt"
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/minio/pkg/console"
	"os"
	"strconv"
)

// TODO: The usage and examples will change as the command implementation evolves after feedback.

var ilmShowCmd = cli.Command{
	Name:   "show",
	Usage:  "Get Information bucket/object lifecycle management information",
	Action: mainLifecycleShow,
	Before: setGlobalsFromContext,
	Flags:  append(ilmShowFlags, globalFlags...),
	CustomHelpTemplate: `Name:
	{{.HelpName}} - {{.Usage}}

USAGE:
 {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
 {{range .VisibleFlags}}{{.}}
 {{end}}
DESCRIPTION:
 ILM show command is to show the user the current lifecycle configuration organized for easy comprehension.

TARGET:
 This argument needs to be in the format of 'alias/bucket/prefix' or 'alias/bucket'

EXAMPLES:
1. Show the lifecycle management rules for the test34bucket on s3. Show all fields.
	{{.Prompt}} {{.HelpName}} s3/test34bucket

2. Show the lifecycle management rules for the test34bucket on s3. Show fields related to expration. Rules with expiration details not set are not shown
	{{.Prompt}} {{.HelpName}} --expiry s3/test34bucket

3. Show the lifecycle management rules for the test34bucket on s3. Show transition details. Rules with transition details not set are not shown
	{{.Prompt}} {{.HelpName}} --transition s3/test34bucket

4. Show the lifecycle management rules for the test34bucket on s3. Minimum details. Mostly if enabled, transition set, expiry set.
	{{.Prompt}} {{.HelpName}} --minimum s3/test34bucket
`,
}

var ilmShowFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "expiry",
		Usage: "Show Expiration Info",
	},
	cli.BoolFlag{
		Name:  "transition",
		Usage: "Show Transition Info",
	},
	cli.BoolFlag{
		Name:  "minimum",
		Usage: "Show Minimum fields",
	},
}

// checkIlmShowSyntax - validate arguments passed by a user
func checkIlmShowSyntax(ctx *cli.Context) {

	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelp(ctx, "show")
		// cli.ShowCommandHelp(ctx, "")
		os.Exit(globalErrorExitStatus)
	}
}

const (
	idLabel             string = "ID"
	prefixLabel         string = "Prefix"
	statusLabel         string = "Enabled"
	statusDisabledLabel string = "Disabled"
	expiryLabel         string = "Expiry"
	expiryDatesLabel    string = "Date/Days"
	singleTagLabel      string = "Tag"
	tagLabel            string = "Tags"
	transitionLabel     string = "Transition"
	transitionDateLabel string = "Date/Days"
	storageClassLabel   string = "Storage-Class"
)

// Get ilm info from alias & bucket
/*func getIlmInfo(urlStr string) (string, *probe.Error) {
	var api *minio.Client
	clientURL := newClientURL(urlStr)
	alias := splitStr(clientURL.String(), "/", 3)[0]
	hostCfg := mustGetHostConfig(alias)

	if hostCfg == nil {
		fatalIf(errInvalidAliasedURL(alias), "No such alias with name `"+alias+"` found.")
	}
	creds := credentials.NewStaticV4(hostCfg.AccessKey, hostCfg.SecretKey, "")
	options := minio.Options{
		Creds:        creds,
		Secure:       false,
		Region:       "",
		BucketLookup: minio.BucketLookupPath,
	}

	hostURLParse, err := url.Parse(hostCfg.URL)
	fatalIf(probe.NewError(err), "Unable to parse `"+hostCfg.URL+"` obtained for alias `"+alias+"`.")

	api, err = minio.NewWithOptions(hostURLParse.Host, &options) // Used as hostname or host:port
	fatalIf(probe.NewError(err), "Unable to connect `"+hostURLParse.Host+"` with obtained credentials.")

	// hostIndex := strings.Index(clientURL.String(), hostname)
	bucketName := splitStr(clientURL.String(), "/", 3)[1]
	if bucketName == "" {
		fatalIf(errInvalidTarget(urlStr), "Given URL:`"+urlStr+"` doesn't have a valid bucket name.")
	}
	lifecycleInfo, err := api.GetBucketLifecycle(bucketName)

	if err != nil {
		return "", probe.NewError(err)
	}

	return lifecycleInfo, nil
}*/

func printIlmInfoRow(info ilmResult, showOpts showDetails) {
	// config := fmt.Sprintf("%-10s: %s", "Configuration", info.XMLName.Local)
	// console.Println(console.Colorize(mainThemeHeader, config))
	if showOpts.json {
		printIlmJSON(info)
		return
	}

	for index := 0; index < len(info.Rules); index++ {
		rule := info.Rules[index]
		expirySet := crossTickCell
		expiryDateSetChk := rule.Expiration != nil && (rule.Expiration.ExpirationDate != nil && !rule.Expiration.ExpirationDate.IsZero())
		expriyDaySetChk := rule.Expiration != nil && (rule.Expiration.ExpirationInDays > 0)
		transitionSetChk := rule.Transition != nil
		transitionDateSetChk := transitionSetChk && (rule.Transition.TransitionDate != nil && !rule.Transition.TransitionDate.IsZero())
		transitionDaySetChk := transitionSetChk && (rule.Transition.TransitionInDays > 0)
		skipExpTran := (showOpts.expiry && !(expiryDateSetChk || expriyDaySetChk)) ||
			(showOpts.transition && !(transitionDateSetChk || transitionDaySetChk))
		if skipExpTran {
			continue
		}
		{
			showInfoFieldFirst(idLabel, rule.ID)
		}
		{
			prefixVal := blankCell
			switch {
			case rule.Prefix != "":
				prefixVal = rule.Prefix
			case rule.RuleFilter.Prefix != "":
				prefixVal = rule.RuleFilter.Prefix
			case rule.RuleFilter.And.Prefix != "":
				prefixVal = rule.RuleFilter.And.Prefix
			}
			showInfoField(prefixLabel, prefixVal)
		}
		{
			statusVal := tickCell
			if rule.Status != statusLabel {
				statusVal = crossTickCell
			}
			showInfoField(statusLabel, statusVal)
		}
		if !showOpts.transition { // Skip expiry section in transition-only display
			showExpDetails := (showOpts.allAvailable || showOpts.expiry) && !showOpts.initial &&
				(expiryDateSetChk || expriyDaySetChk)
			if !showOpts.expiry {
				expirySet = tickCell
				showInfoField(expiryLabel, expirySet)
			}
			if showExpDetails {
				expiryDate := blankCell
				if expiryDateSetChk {
					expiryDate = strconv.Itoa(rule.Expiration.ExpirationDate.Day()) + " " +
						rule.Expiration.ExpirationDate.Month().String()[0:3] + " " +
						strconv.Itoa(rule.Expiration.ExpirationDate.Year())
				} else if expriyDaySetChk {
					expiryDate = strconv.Itoa(rule.Expiration.ExpirationInDays) + " day(s)"
				}
				showInfoField(expiryDatesLabel, expiryDate)
			}
		}
		if !showOpts.expiry { // Skip transition section in expiry-only display
			transitionSet := crossTickCell
			storageSetChk := transitionSetChk && (rule.Transition.StorageClass != "")
			if transitionDateSetChk || transitionDaySetChk {
				transitionSet = tickCell
			}
			if !showOpts.transition {
				showInfoField(transitionLabel, transitionSet)
			}
			showTransitionDetails := (showOpts.allAvailable || showOpts.transition) && !showOpts.initial && transitionSet != crossTickCell
			if showTransitionDetails {
				transitionDate := blankCell
				storageClass := blankCell
				if transitionDateSetChk {
					transitionDate = strconv.Itoa(rule.Transition.TransitionDate.Day()) + " " +
						rule.Transition.TransitionDate.Month().String()[0:3] + " " +
						strconv.Itoa(rule.Transition.TransitionDate.Year())
				} else if transitionDaySetChk {
					transitionDate = strconv.Itoa(rule.Transition.TransitionInDays) + " day(s)"
				}
				if storageSetChk {
					storageClass = rule.Transition.StorageClass
				}
				showInfoField(transitionDateLabel, transitionDate)
				showInfoField(storageClassLabel, storageClass)
			}
		}
		{
			tagArr := rule.TagFilters
			tagLth := len(rule.TagFilters)
			andTagsSetChk := len(rule.TagFilters) == 0 && rule.RuleFilter != nil && rule.RuleFilter.And != nil &&
				len(rule.RuleFilter.And.Tags) > 0
			if andTagsSetChk {
				tagLth = len(rule.RuleFilter.And.Tags)
				tagArr = rule.RuleFilter.And.Tags
			}
			tagShowArr := []string{}
			for tagIdx := 0; tagIdx < tagLth; tagIdx++ {
				tagShowArr = append(tagShowArr, tagArr[tagIdx].Key+":"+tagArr[tagIdx].Value)
			}
			if len(tagArr) > 0 {
				showInfoFieldMultiple(tagLabel, tagShowArr)
			}
		}
	}
}
func setColorScheme() {
	console.SetColor(fieldMainHeader, color.New(color.Bold, color.FgHiRed))
	console.SetColor(fieldThemeRow, color.New(color.FgHiWhite))
	console.SetColor(fieldThemeHeader, color.New(color.FgCyan))
	console.SetColor(fieldThemeTick, color.New(color.FgGreen))
	console.SetColor(fieldThemeExpiry, color.New(color.BlinkRapid, color.FgGreen))
	console.SetColor(fieldThemeResultSuccess, color.New(color.FgGreen, color.Bold))
}

func getShowOpts(ctx *cli.Context) showDetails {
	showOpts := showDetails{
		allAvailable: true,
		expiry:       ctx.Bool("expiry"),
		transition:   ctx.Bool("transition"),
		json:         ctx.Bool("json"),
		initial:      ctx.Bool("minimum"),
	}
	if showOpts.expiry || showOpts.transition || showOpts.initial {
		showOpts.allAvailable = false
	} else if showOpts.allAvailable {
		showOpts.expiry = false
		showOpts.transition = false
		showOpts.initial = false
	}
	return showOpts
}

func mainLifecycleShow(ctx *cli.Context) error {
	checkIlmShowSyntax(ctx)
	setColorScheme()
	args := ctx.Args()
	objectURL := args.Get(0)
	lfcInfoXML, err := getIlmInfo(objectURL)
	if err != nil {
		fmt.Println(err)
		return err.ToGoError()
	}
	if lfcInfoXML == "" {
		return nil
	}
	// fmt.Println(lfcInfoXML)
	lfcInfo := ilmResult{}
	err2 := xml.Unmarshal([]byte(lfcInfoXML), &lfcInfo)
	if err2 != nil {
		fmt.Println(err2)
		return err2
	}
	showOpts := getShowOpts(ctx)
	printIlmInfoRow(lfcInfo, showOpts)
	return nil
}
