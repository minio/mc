/*
 * Minio Client (C) 2017 Minio, Inc.
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
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var adminHealFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "recursive, r",
		Usage: "Heal recursively",
	},
	cli.BoolFlag{
		Name:  "dry-run, n",
		Usage: "Only inspect data, but do not mutate",
	},
	cli.BoolFlag{
		Name:  "incomplete, I",
		Usage: "Heal uploads recursively",
	},
	cli.BoolFlag{
		Name:  "force-start, f",
		Usage: "Force start a heal sequence",
	},
}

var adminHealCmd = cli.Command{
	Name:  "heal",
	Usage: "Start an object heal operation",
	// Before:          adminHealBefore,
	Action:          mainAdminHeal,
	Flags:           append(adminHealFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    1. To format newly replaced disks in a Minio server with alias 'play'
       $ {{.HelpName}} play

    2. Heal 'testbucket' in a Minio server with alias 'play'
       $ {{.HelpName}} play/testbucket/

    3. Heal all objects under 'dir' prefix
       $ {{.HelpName}} --recursive play/testbucket/dir/

    4. Heal all objects including uploads under 'dir' prefix
       $ {{.HelpName}} --incomplete --recursive play/testbucket/dir/

    5. Issue a dry-run heal operation to list all objects to be healed
       $ {{.HelpName}} --dry-run play

    6. Issue a dry-run heal operation to list all objects to be healed under 'dir' prefix
       $ {{.HelpName}} --recursive --dry-run play/testbucket/dir/

    7. Force start a heal operation (in case another heal operation is running)
       $ {{.HelpName}} --recursive --force-start play/testbucket/dir/

`,
}

const (
	lineWidth = 80
)

// An alias of string to represent the health color code of an object
type hCol string

const (
	hColGrey   hCol = "Grey"
	hColRed         = "Red"
	hColYellow      = "Yellow"
	hColGreen       = "Green"
)

// getHPrintCol - map color code to color
func getHPrintCol(c hCol) *color.Color {
	switch c {
	case hColGrey:
		return color.New(color.FgWhite, color.Bold)
	case hColRed:
		return color.New(color.FgRed, color.Bold)
	case hColYellow:
		return color.New(color.FgYellow, color.Bold)
	case hColGreen:
		return color.New(color.FgGreen, color.Bold)
	}
	return nil
}

var (
	hColOrder = []hCol{hColRed, hColYellow, hColGreen}
	hColTable = map[int][]int{
		1: {0, -1, 1},
		2: {0, 1, 2},
		3: {1, 2, 3},
		4: {1, 2, 4},
		5: {1, 3, 5},
		6: {2, 4, 6},
		7: {2, 4, 7},
		8: {2, 5, 8},
	}
)

func getHColCode(surplusShards, parityShards int) (c hCol, err error) {
	if parityShards < 1 || parityShards > 8 || surplusShards > parityShards {
		return c, fmt.Errorf("Invalid parity shard count/surplus shard count given")
	}
	if surplusShards < 0 {
		return hColGrey, err
	}
	colRow := hColTable[parityShards]
	for index, val := range colRow {
		if val != -1 && surplusShards <= val {
			return hColOrder[index], err
		}
	}
	return c, fmt.Errorf("this will not be returned")
}

type hri struct {
	*madmin.HealResultItem
}

func newHRI(i *madmin.HealResultItem) *hri {
	return &hri{i}
}

// getObjectHCCChange - returns before and after color change for
// objects
func (h hri) getObjectHCCChange() (b, a hCol, err error) {
	parityShards := h.ParityBlocks
	dataShards := h.DataBlocks

	onlineBefore, onlineAfter := h.GetOnlineCounts()
	surplusShardsBeforeHeal := onlineBefore - dataShards
	surplusShardsAfterHeal := onlineAfter - dataShards

	b, err = getHColCode(surplusShardsBeforeHeal, parityShards)
	if err != nil {
		return
	}
	a, err = getHColCode(surplusShardsAfterHeal, parityShards)
	return

}

// getReplicatedFileHCCChange - fetches health color code for metadata
// files that are replicated.
func (h hri) getReplicatedFileHCCChange() (b, a hCol, err error) {
	getColCode := func(numAvail int) (c hCol, err error) {
		// calculate color code for replicated object similar
		// to erasure coded objects
		quorum := h.DiskCount/2 + 1
		surplus := numAvail - quorum
		parity := h.DiskCount - quorum
		c, err = getHColCode(surplus, parity)
		return
	}

	onlineBefore, onlineAfter := h.GetOnlineCounts()
	b, err = getColCode(onlineBefore)
	if err != nil {
		return
	}
	a, err = getColCode(onlineAfter)
	return
}

func (h hri) makeHealEntityString() string {
	switch h.Type {
	case madmin.HealItemObject:
		return h.Bucket + "/" + h.Object
	case madmin.HealItemBucket:
		return h.Bucket
	case madmin.HealItemMetadata:
		return "[disk-format]"
	case madmin.HealItemBucketMetadata:
		return fmt.Sprintf("[bucket-metadata]%s/%s", h.Bucket, h.Object)
	case madmin.HealItemMultipartUpload:
		return fmt.Sprintf("[multipart-upload]%s/%s(%s)", h.Bucket,
			h.Object, h.Detail)
	}
	return "**unexpected**"
}

func (h hri) getHRTypeAndName() (typ, name string) {
	name = fmt.Sprintf("%s/%s", h.Bucket, h.Object)
	switch h.Type {
	case madmin.HealItemMetadata:
		typ = "system"
		name = h.Detail
	case madmin.HealItemBucketMetadata:
		typ = "system"
		name = "bucket-metadata:" + name
	case madmin.HealItemMultipartUpload:
		typ = "object"
		name = "ongoing-upload:" + name
	case madmin.HealItemBucket:
		typ = "bucket"
	case madmin.HealItemObject:
		typ = "object"
	default:
		typ = fmt.Sprintf("!! Unknown heal result record %#v !!", h)
		name = typ
	}
	return
}

func (h hri) getHealResultStr() string {
	typ, name := h.getHRTypeAndName()

	switch h.Type {
	case madmin.HealItemMetadata, madmin.HealItemBucketMetadata:
		return typ + ":" + name
	default:
		return name
	}
}

// lineTrunc - truncates a string by adding ellipsis in the middle
func lineTrunc(content string, maxLen int) string {
	runes := []rune(content)
	rlen := len(runes)
	if rlen <= maxLen {
		return content
	}
	halfLen := maxLen / 2
	fstPart := string(runes[0:halfLen])
	sndPart := string(runes[rlen-halfLen:])
	return fstPart + "…" + sndPart
}

type uiData struct {
	Bucket, Prefix string
	Client         *madmin.AdminClient
	ClientToken    string
	ForceStart     bool
	HealOpts       *madmin.HealOpts
	LastItem       *hri
	Quiet          bool
	JSON           bool

	// Total time since heal start
	HealDuration time.Duration

	// Accumulated statistics of heal result records
	BytesScanned   int64
	ObjectsScanned int64
	ObjectsHealed  int64
	// Map from online drives to number of objects with that many
	// online drives.
	ObjectsByOnlineDrives map[int]int64
	// Map of health color code to number of objects with that
	// health color code.
	HealthCols map[hCol]int64
}

func (ui *uiData) updateStats(i madmin.HealResultItem) error {
	// Objects whose size could not be found have -1 size
	// returned.
	if i.ObjectSize >= 0 {
		ui.BytesScanned += i.ObjectSize
	}

	ui.ObjectsScanned++
	beforeUp, afterUp := i.GetOnlineCounts()
	if afterUp > beforeUp {
		ui.ObjectsHealed++
	}
	ui.ObjectsByOnlineDrives[afterUp]++

	// Update health color stats:

	// Fetch health color after heal:
	var err error
	var afterCol hCol
	h := newHRI(&i)
	switch h.Type {
	case madmin.HealItemMetadata, madmin.HealItemBucket:
		_, afterCol, err = h.getReplicatedFileHCCChange()
	default:
		_, afterCol, err = h.getObjectHCCChange()
	}
	if err != nil {
		fmt.Println("AAAARGGH!", err)
		return err
	}

	ui.HealthCols[afterCol]++
	return nil
}

func (ui *uiData) updateDuration(s *madmin.HealTaskStatus) {
	ui.HealDuration = UTCNow().Sub(s.StartTime)
}

func (ui *uiData) getProgress() (oCount, objSize, duration string) {
	oCount = humanize.Comma(ui.ObjectsScanned)

	duration = ui.HealDuration.Round(time.Second).String()

	bytesScanned := float64(ui.BytesScanned)

	// Compute unit for object size
	magnitudes := []float64{1 << 10, 1 << 20, 1 << 30, 1 << 40, 1 << 50, 1 << 60}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	var i int
	for i = 0; i < len(magnitudes); i++ {
		if bytesScanned <= magnitudes[i] {
			break
		}
	}
	numUnits := int(bytesScanned * (1 << 10) / magnitudes[i])
	objSize = fmt.Sprintf("%d %s", numUnits, units[i])
	return
}

func (ui *uiData) getPercentsNBars() (p map[hCol]float64, b map[hCol]string) {
	// barChar, emptyBarChar := "█", "░"
	barChar, emptyBarChar := "█", " "
	barLen := 12
	sum := float64(ui.ObjectsScanned)
	cols := []hCol{hColGrey, hColRed, hColYellow, hColGreen}

	p = make(map[hCol]float64, len(cols))
	b = make(map[hCol]string, len(cols))
	var filledLen int
	for _, col := range cols {
		v := float64(ui.HealthCols[col])
		if sum == 0 {
			p[col] = 0
			filledLen = 0
		} else {
			p[col] = v * 100 / sum
			// round up the filled part
			filledLen = int(math.Ceil(float64(barLen) * v / sum))
		}
		b[col] = strings.Repeat(barChar, filledLen) +
			strings.Repeat(emptyBarChar, barLen-filledLen)
	}
	return
}

func printItemsQuietly(s *madmin.HealTaskStatus) (err error) {
	lpad := func(s hCol) string {
		return fmt.Sprintf("%-6s", string(s))
	}
	rpad := func(s hCol) string {
		return fmt.Sprintf("%6s", string(s))
	}
	printColStr := func(before, after hCol) {
		console.PrintC("[" + lpad(before) + " -> " + rpad(after) + "] ")
	}

	var b, a hCol
	for _, item := range s.Items {
		h := newHRI(&item)
		switch h.Type {
		case madmin.HealItemMetadata, madmin.HealItemBucket:
			b, a, err = h.getReplicatedFileHCCChange()
		default:
			b, a, err = h.getObjectHCCChange()
		}
		if err != nil {
			return err
		}
		printColStr(b, a)
		hrStr := h.getHealResultStr()
		switch h.Type {
		case madmin.HealItemMetadata, madmin.HealItemBucketMetadata:
			console.PrintC(fmt.Sprintln("** ", hrStr, " **"))
		default:
			console.PrintC(hrStr, "\n")
		}
	}
	return nil
}

func (ui *uiData) printStatsQuietly(s *madmin.HealTaskStatus) {
	console.PrintC("\nSummary:\n")

	totalObjects, totalSize, totalTime := ui.getProgress()

	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "    Objects Scanned:\t%s\n", totalObjects)
	fmt.Fprintf(w, "    Objects Healed:\t%s\n", humanize.Comma(ui.ObjectsHealed))
	fmt.Fprintf(w, "    Size:\t%s\n", totalSize)
	fmt.Fprintf(w, "    Elapsed Time:\t%s\n", totalTime)
	w.Flush()

	console.PrintC(b.String())
}

func printItemsJSON(s *madmin.HealTaskStatus) (err error) {
	type change struct {
		Before string `json:"before"`
		After  string `json:"after"`
	}
	type healRec struct {
		Status string            `json:"status"`
		Error  string            `json:"error,omitempty"`
		Type   string            `json:"type"`
		Name   string            `json:"name"`
		Health change            `json:"health"`
		Drives map[string]change `json:"drives"`
		Size   int64             `json:"size"`
	}
	makeHR := func(h *hri) (r healRec, err error) {
		r.Status = "success"
		r.Type, r.Name = h.getHRTypeAndName()
		r.Drives = make(map[string]change)

		var b, a hCol
		switch h.Type {
		case madmin.HealItemMetadata, madmin.HealItemBucket:
			b, a, err = h.getReplicatedFileHCCChange()
		default:
			if h.Type == madmin.HealItemObject {
				r.Size = h.ObjectSize
			}
			b, a, err = h.getObjectHCCChange()
		}
		if err != nil {
			return r, err
		}
		r.Health.Before = strings.ToLower(string(b))
		r.Health.After = strings.ToLower(string(a))

		for k := range h.DriveInfo.Before {
			r.Drives[k] = change{h.DriveInfo.Before[k], h.DriveInfo.After[k]}
		}

		return r, nil
	}

	for _, item := range s.Items {
		h := newHRI(&item)
		r, err := makeHR(h)
		if err != nil {
			return err
		}
		jsonBytes, err := json.Marshal(r)
		fatalIf(probe.NewError(err), "Unable to marshal to JSON")
		console.Println(string(jsonBytes))
	}
	return nil
}

func (ui *uiData) printStatsJSON(s *madmin.HealTaskStatus) {
	var summary struct {
		Status         string `json:"status"`
		Error          string `json:"error,omitempty"`
		Type           string `json:"type"`
		ObjectsScanned int64  `json:"objects_scanned"`
		ObjectsHealed  int64  `json:"objects_healed"`
		Size           int64  `json:"size_bytes"`
		ElapsedTime    int64  `json:"elapsed_time_seconds"`
	}

	summary.Status = "success"
	summary.Type = "summary"

	summary.ObjectsScanned = ui.ObjectsScanned
	summary.ObjectsHealed = ui.ObjectsHealed
	summary.Size = ui.BytesScanned
	summary.ElapsedTime = int64(ui.HealDuration.Round(time.Second).Seconds())

	jBytes, err := json.Marshal(summary)
	fatalIf(probe.NewError(err), "Unable to marshal to JSON")
	console.Println(string(jBytes))
}

func (ui *uiData) updateUI(s *madmin.HealTaskStatus) (err error) {
	itemCount := len(s.Items)
	h := ui.LastItem
	if itemCount > 0 {
		item := s.Items[itemCount-1]
		h = newHRI(&item)
		ui.LastItem = h
	}
	scannedStr := "** waiting for status from server **"
	if h != nil {
		scannedStr = lineTrunc(h.makeHealEntityString(), lineWidth-len("Scanned: "))
	}

	totalObjects, totalSize, totalTime := ui.getProgress()
	completedStr := fmt.Sprintf("%s objects, %s in %s",
		totalObjects, totalSize, totalTime)

	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, 0, 0, 1, ' ', 0)
	// Object:
	fmt.Fprintf(w, "Object:\t%s\n", scannedStr)
	fmt.Fprintf(w, "Scanned:\t%s\n", completedStr)
	fmt.Fprintf(w, "Healed:\t%s\n\n", humanize.Comma(ui.ObjectsHealed))
	w.Flush()

	console.PrintC(b.String())

	console.PrintC("System Health:\n")

	dspOrder := []hCol{hColGreen, hColYellow, hColRed, hColGrey}
	printColors := []*color.Color{}
	for _, c := range dspOrder {
		printColors = append(printColors, getHPrintCol(c))
	}
	t := console.NewTable("", printColors,
		[]bool{false, true, true, false},
	)

	percentMap, barMap := ui.getPercentsNBars()
	cellText := make([][]string, len(dspOrder))
	for i := range cellText {
		cellText[i] = []string{
			string(dspOrder[i]),
			fmt.Sprintf("%.1f%%", percentMap[dspOrder[i]]),
			fmt.Sprintf(humanize.Comma(ui.HealthCols[dspOrder[i]])),
			barMap[dspOrder[i]],
		}
	}

	t.DisplayTable(cellText)
	return nil
}

func (ui *uiData) UpdateDisplay(s *madmin.HealTaskStatus) (err error) {
	// Update state
	ui.updateDuration(s)
	for _, i := range s.Items {
		ui.updateStats(i)
	}

	// Update display
	switch {
	case ui.JSON:
		err = printItemsJSON(s)
	case ui.Quiet:
		err = printItemsQuietly(s)
	default:
		err = ui.updateUI(s)
	}
	return
}

func (ui *uiData) DisplayAndFollowHealStatus() (err error) {
	var res madmin.HealTaskStatus

	for {
		_, res, err = ui.Client.Heal(ui.Bucket, ui.Prefix, *ui.HealOpts,
			ui.ClientToken, ui.ForceStart)
		if err != nil {
			return err
		}

		err = ui.UpdateDisplay(&res)
		if err != nil {
			return err
		}

		if res.Summary == "finished" {
			break
		}

		if res.Summary == "stopped" {
			fmt.Println("Heal had an error - ", res.FailureDetail)
			break
		}

		time.Sleep(time.Second)
		if !ui.JSON && !ui.Quiet {
			console.RewindLines(11)
		}
	}
	if ui.JSON {
		ui.printStatsJSON(&res)
		return nil
	}
	if ui.Quiet {
		ui.printStatsQuietly(&res)
	}
	return nil
}

// adminHealBefore used to provide users with temporary warning message
func adminHealBefore(ctx *cli.Context) error {
	color.Yellow("\t *** mc admin heal is EXPERIMENTAL ***")
	return setGlobalsFromContext(ctx)
}

func checkAdminHealSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "heal", 1) // last argument is exit code
	}
}

// mainAdminHeal - the entry function of heal command
func mainAdminHeal(ctx *cli.Context) error {

	// Check for command syntax
	checkAdminHealSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	console.SetColor("Heal", color.New(color.FgGreen, color.Bold))

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Cannot initialize admin client.")
		return nil
	}

	// Compute bucket and object from the aliased URL
	aliasedURL = filepath.ToSlash(aliasedURL)
	splits := splitStr(aliasedURL, "/", 3)
	bucket, prefix := splits[1], splits[2]

	opts := madmin.HealOpts{
		Recursive:  ctx.Bool("recursive"),
		DryRun:     ctx.Bool("dry-run"),
		Incomplete: ctx.Bool("incomplete"),
	}
	forceStart := ctx.Bool("force-start")
	healStart, _, herr := client.Heal(bucket, prefix, opts, "", forceStart)
	errorIf(probe.NewError(herr), "Failed to start heal sequence")

	ui := uiData{
		Bucket:      bucket,
		Prefix:      prefix,
		Client:      client,
		ClientToken: healStart.ClientToken,
		ForceStart:  forceStart,
		HealOpts:    &opts,
		Quiet:       ctx.Bool("quiet"),
		JSON:        ctx.Bool("json"),
		ObjectsByOnlineDrives: make(map[int]int64),
		HealthCols:            make(map[hCol]int64),
	}
	errorIf(
		probe.NewError(ui.DisplayAndFollowHealStatus()),
		"Unable to display follow heal status",
	)

	return nil
}
