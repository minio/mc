// Copyright (c) 2022 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/fatih/color"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var replicateBacklogFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "arn",
		Usage: "unique role ARN",
	},
	cli.BoolFlag{
		Name:  "verbose,v",
		Usage: "include replicated versions",
	},
	cli.StringFlag{
		Name:  "nodes,n",
		Usage: "show most recent failures for one or more nodes. Valid values are 'all', or node name",
		Value: "all",
	},
	cli.BoolFlag{
		Name:  "full,a",
		Usage: "list and show all replication failures for bucket",
	},
}

var replicateBacklogCmd = cli.Command{
	Name:          "backlog",
	Aliases:       []string{"diff"},
	HiddenAliases: true,
	Usage:         "show unreplicated object versions",
	Action:        mainReplicateBacklog,
	OnUsageError:  onUsageError,
	Before:        setGlobalsFromContext,
	Flags:         append(globalFlags, replicateBacklogFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Show most recent replication failures on "myminio" alias for objects in bucket "mybucket"
     {{.Prompt}} {{.HelpName}} myminio/mybucket

  2. Show all unreplicated objects on "myminio" alias for objects in prefix "path/to/prefix" of "mybucket" for all targets.
     This will perform full listing of all objects in the prefix to find unreplicated objects.
     {{.Prompt}} {{.HelpName}} myminio/mybucket/path/to/prefix --full
`,
}

// checkReplicateBacklogSyntax - validate all the passed arguments
func checkReplicateBacklogSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

type replicateMRFMessage struct {
	Op     string `json:"op"`
	Status string `json:"status"`
	madmin.ReplicationMRF
}

func (m replicateMRFMessage) JSON() string {
	m.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (m replicateMRFMessage) String() string {
	return console.Colorize("", newPrettyTable(" | ",
		Field{getNodeTheme(m.ReplicationMRF.NodeName), len(m.ReplicationMRF.NodeName) + 3},
		Field{"Count", 7},
		Field{"Object", -1},
	).buildRow(m.NodeName, fmt.Sprintf("Retry=%d", m.RetryCount), fmt.Sprintf("%s (%s)", m.Object, m.VersionID)))
}

type replicateBacklogMessage struct {
	Op       string                `json:"op"`
	Diff     madmin.DiffInfo       `json:"diff,omitempty"`
	MRF      madmin.ReplicationMRF `json:"mrf,omitempty"`
	OpStatus string                `json:"opStatus"`
	arn      string                `json:"-"`
	verbose  bool                  `json:"-"`
}

func (r replicateBacklogMessage) JSON() string {
	var e error
	var jsonMessageBytes []byte
	switch r.Op {
	case "diff":
		jsonMessageBytes, e = json.MarshalIndent(r.Diff, "", " ")

	case "mrf":
		jsonMessageBytes, e = json.MarshalIndent(r.MRF, "", " ")
	}
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (r replicateBacklogMessage) toRow() (row table.Row) {
	switch r.Op {
	case "diff":
		return r.toDiffRow()
	case "mrf":
		return r.toMRFRow()
	}
	return
}

func (r replicateBacklogMessage) toDiffRow() (row table.Row) {
	d := r.Diff
	if d.Object == "" {
		return
	}
	op := ""
	if d.VersionID != "" {
		switch d.IsDeleteMarker {
		case true:
			op = "DEL"
		default:
			op = "PUT"
		}
	}
	st := r.replStatus()
	replTimeStamp := d.ReplicationTimestamp.Format(printDate)
	switch {
	case st == "PENDING":
		replTimeStamp = ""
	case op == "DEL":
		replTimeStamp = ""
	}
	return table.Row{
		replTimeStamp, d.LastModified.Format(printDate), st, d.VersionID, op, d.Object,
	}
}

func (r replicateBacklogMessage) toMRFRow() (row table.Row) {
	d := r.MRF
	if d.Object == "" {
		return
	}
	return table.Row{
		d.NodeName, d.VersionID, strconv.Itoa(d.RetryCount), path.Join(d.Bucket, d.Object),
	}
}

func (r *replicateBacklogMessage) replStatus() string {
	var st string
	d := r.Diff
	if r.arn == "" { // report overall replication status
		if d.DeleteReplicationStatus != "" {
			st = d.DeleteReplicationStatus
		} else {
			st = d.ReplicationStatus
		}
	} else { // report target replication diff
		for arn, t := range d.Targets {
			if arn != r.arn {
				continue
			}
			if t.DeleteReplicationStatus != "" {
				st = t.DeleteReplicationStatus
			} else {
				st = t.ReplicationStatus
			}
		}
		if len(d.Targets) == 0 {
			st = ""
		}
	}
	return st
}

type replicateBacklogUI struct {
	spinner  spinner.Model
	sub      any
	diffCh   chan madmin.DiffInfo
	mrfCh    chan madmin.ReplicationMRF
	arn      string
	op       string
	quitting bool
	table    table.Model
	rows     []table.Row
	help     help.Model
	keymap   keyMap
	count    int
}
type keyMap struct {
	quit  key.Binding
	up    key.Binding
	down  key.Binding
	enter key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		up: key.NewBinding(
			key.WithKeys("k", "up", "left", "shift+tab"),
			key.WithHelp("↑/k", "Move up"),
		),
		down: key.NewBinding(
			key.WithKeys("j", "down", "right", "tab"),
			key.WithHelp("↓/j", "Move down"),
		),
		enter: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/spacebar", ""),
		),
		quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("q", "quit"),
		),
	}
}

func initReplicateBacklogUI(arn, op string, diffCh any) *replicateBacklogUI {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	columns := getBacklogHeader(op)

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	ts := getBacklogStyles()
	t.SetStyles(ts)

	ui := &replicateBacklogUI{
		spinner: s,
		sub:     diffCh,
		op:      op,
		arn:     arn,
		table:   t,
		help:    help.New(),
		keymap:  newKeyMap(),
	}
	if ch, ok := diffCh.(chan madmin.DiffInfo); ok {
		ui.diffCh = ch
	}
	if ch, ok := diffCh.(chan madmin.ReplicationMRF); ok {
		ui.mrfCh = ch
	}
	return ui
}

func (m *replicateBacklogUI) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		waitForActivity(m.sub, m.op), // wait for activity
	)
}

const rowLimit = 10000

// A command that waits for the activity on a channel.
func waitForActivity(sub any, op string) tea.Cmd {
	return func() tea.Msg {
		switch op {
		case "diff":
			msg := <-sub.(<-chan madmin.DiffInfo)
			return msg
		case "mrf":
			msg := <-sub.(<-chan madmin.ReplicationMRF)
			return msg
		}
		return "unexpected message"
	}
}

func getBacklogHeader(op string) []table.Column {
	switch op {
	case "diff":
		return getBacklogDiffHeader()
	case "mrf":
		return getBacklogMRFHeader()
	}
	return nil
}

func getBacklogDiffHeader() []table.Column {
	return []table.Column{
		{Title: "Attempted At", Width: 23},
		{Title: "Created", Width: 23},
		{Title: "Status", Width: 9},
		{Title: "VersionID", Width: 36},
		{Title: "Op", Width: 3},
		{Title: "Object", Width: 60},
	}
}

func getBacklogMRFHeader() []table.Column {
	return []table.Column{
		{Title: "Node", Width: 40},
		{Title: "VersionID", Width: 36},
		{Title: "Retry", Width: 5},
		{Title: "Object", Width: 60},
	}
}

func getBacklogStyles() table.Styles {
	ts := table.DefaultStyles()
	ts.Header = ts.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	ts.Selected = ts.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("300")).
		Bold(false)
	return ts
}

func (m *replicateBacklogUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			columns := getBacklogHeader(m.op)
			ts := getBacklogStyles()
			m.table = table.New(
				table.WithColumns(columns),
				table.WithRows(m.rows),
				table.WithFocused(true),
				table.WithHeight(10),
			)
			m.table.SetStyles(ts)
		default:
		}
	case madmin.DiffInfo:
		if msg.Object != "" {
			m.count++
			if m.count <= rowLimit { // don't buffer more than 10k entries
				rdif := replicateBacklogMessage{
					Op:   "diff",
					Diff: msg,
					arn:  m.arn,
				}
				m.rows = append(m.rows, rdif.toRow())
			}
			return m, waitForActivity(m.sub, m.op)
		}
		m.quitting = true
		columns := getBacklogDiffHeader()
		ts := getBacklogStyles()
		m.table = table.New(
			table.WithColumns(columns),
			table.WithRows(m.rows),
			table.WithFocused(true),
			table.WithHeight(10),
		)
		m.table.SetStyles(ts)
		return m, nil
	case madmin.ReplicationMRF:
		if msg.Object != "" {
			m.count++
			if m.count <= rowLimit { // don't buffer more than 10k entries
				rdif := replicateBacklogMessage{
					Op:  "mrf",
					MRF: msg,
					arn: m.arn,
				}
				m.rows = append(m.rows, rdif.toRow())
			}
			return m, waitForActivity(m.sub, m.op)
		}
		m.quitting = true
		columns := getBacklogMRFHeader()
		ts := getBacklogStyles()
		m.table = table.New(
			table.WithColumns(columns),
			table.WithRows(m.rows),
			table.WithFocused(true),
			table.WithHeight(10),
		)
		m.table.SetStyles(ts)
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		if !m.quitting {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	m.table, cmd = m.table.Update(msg)

	return m, cmd
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var descStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
	Light: "#B2B2B2",
	Dark:  "#4A4A4A",
})

var (
	subtle  = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	special = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	divider = lipgloss.NewStyle().
		SetString("•").
		Padding(0, 1).
		Foreground(subtle).
		String()

	advisory  = lipgloss.NewStyle().Foreground(special).Render
	infoStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(subtle)
)

func (m *replicateBacklogUI) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.enter,
		m.keymap.down,
		m.keymap.up,
		m.keymap.quit,
	})
}

func (m *replicateBacklogUI) View() string {
	var sb strings.Builder
	if !m.quitting {
		sb.WriteString(fmt.Sprintf("%s\n", m.spinner.View()))
	}

	if m.count > 0 {
		advisoryStr := ""
		if m.count > rowLimit {
			advisoryStr = "[ use --json flag for full listing]"
		}
		desc := lipgloss.JoinVertical(lipgloss.Left,
			descStyle.Render("Unreplicated versions summary"),
			infoStyle.Render(fmt.Sprintf("Total Unreplicated: %d", m.count)+divider+advisory(advisoryStr+"\n")))
		row := lipgloss.JoinHorizontal(lipgloss.Top, desc)
		sb.WriteString(row + "\n\n")
		sb.WriteString(baseStyle.Render(m.table.View()))
	}
	sb.WriteString(m.helpView())

	return sb.String()
}

func mainReplicateBacklog(cliCtx *cli.Context) error {
	checkReplicateBacklogSyntax(cliCtx)
	console.SetColor("diff-msg", color.New(color.FgHiCyan, color.Bold))
	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	aliasedURL = filepath.ToSlash(aliasedURL)
	splits := splitStr(aliasedURL, "/", 3)
	bucket, prefix := splits[1], splits[2]
	if bucket == "" {
		fatalIf(errInvalidArgument(), "bucket not specified in `"+aliasedURL+"`.")
	}
	ctx, cancel := context.WithCancel(globalContext)
	defer cancel()

	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")
	if !cliCtx.Bool("full") {
		mrfCh := client.BucketReplicationMRF(ctx, bucket, cliCtx.String("nodes"))
		if globalJSON {
			for mrf := range mrfCh {
				if mrf.Err != "" {
					fatalIf(probe.NewError(fmt.Errorf("%s", mrf.Err)), "Unable to fetch replication backlog.")
				}
				printMsg(replicateMRFMessage{
					Op:             "mrf",
					Status:         "success",
					ReplicationMRF: mrf,
				})
			}
			return nil
		}
		ui := tea.NewProgram(initReplicateBacklogUI("", "mrf", mrfCh))
		if _, e := ui.Run(); e != nil {
			cancel()
			fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to fetch replication backlog")
		}
		return nil
	}

	verbose := cliCtx.Bool("verbose")
	arn := cliCtx.String("arn")
	diffCh := client.BucketReplicationDiff(ctx, bucket, madmin.ReplDiffOpts{
		Verbose: verbose,
		ARN:     arn,
		Prefix:  prefix,
	})
	if globalJSON {
		for di := range diffCh {
			console.Println(replicateBacklogMessage{
				Op:      "diff",
				Diff:    di,
				arn:     arn,
				verbose: verbose,
			}.JSON())
		}
		return nil
	}

	ui := tea.NewProgram(initReplicateBacklogUI(arn, "diff", diffCh))
	if _, e := ui.Run(); e != nil {
		cancel()
		fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to fetch replication backlog")
	}

	return nil
}
