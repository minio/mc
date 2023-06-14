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
	"path/filepath"
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
	"github.com/minio/pkg/console"
)

var replicateDiffFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "arn",
		Usage: "unique role ARN",
	},
	cli.BoolFlag{
		Name:  "verbose,v",
		Usage: "include replicated versions",
	},
}

var replicateDiffCmd = cli.Command{
	Name:         "diff",
	Usage:        "show unreplicated object versions",
	Action:       mainReplicateDiff,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, replicateDiffFlags...),
	CustomHelpTemplate: `NAME:
   {{.HelpName}} - {{.Usage}}

USAGE:
   {{.HelpName}} TARGET

FLAGS:
   {{range .VisibleFlags}}{{.}}
   {{end}}
EXAMPLES:
  1. Show unreplicated objects on "myminio" alias for objects in prefix "path/to/prefix" of "mybucket" for a specific remote target
     {{.Prompt}} {{.HelpName}} myminio/mybucket/path/to/prefix --arn <remote-arn>

  2. Show unreplicated objects on "myminio" alias for objects in prefix "path/to/prefix" of "mybucket" for all targets.
     {{.Prompt}} {{.HelpName}} myminio/mybucket/path/to/prefix
`,
}

// checkReplicateDiffSyntax - validate all the passed arguments
func checkReplicateDiffSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

type replicateDiffMessage struct {
	Op string `json:"op"`
	madmin.DiffInfo
	OpStatus string `json:"opStatus"`
	arn      string `json:"-"`
	verbose  bool   `json:"-"`
}

func (r replicateDiffMessage) JSON() string {
	r.OpStatus = "success"
	jsonMessageBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (r replicateDiffMessage) toRow() (row table.Row) {
	if r.Object == "" {
		return
	}
	op := ""
	if r.VersionID != "" {
		switch r.IsDeleteMarker {
		case true:
			op = "DEL"
		default:
			op = "PUT"
		}
	}
	st := r.replStatus()
	replTimeStamp := r.ReplicationTimestamp.Format(printDate)
	switch {
	case st == "PENDING":
		replTimeStamp = ""
	case op == "DEL":
		replTimeStamp = ""
	}
	return table.Row{
		replTimeStamp, r.LastModified.Format(printDate), st, r.VersionID, op, r.Object,
	}
}

func (r *replicateDiffMessage) replStatus() string {
	var st string
	if r.arn == "" { // report overall replication status
		if r.DeleteReplicationStatus != "" {
			st = r.DeleteReplicationStatus
		} else {
			st = r.ReplicationStatus
		}
	} else { // report target replication diff
		for arn, t := range r.Targets {
			if arn != r.arn {
				continue
			}
			if t.DeleteReplicationStatus != "" {
				st = t.DeleteReplicationStatus
			} else {
				st = t.ReplicationStatus
			}
		}
		if len(r.Targets) == 0 {
			st = ""
		}
	}
	return st
}

const rowLimit = 10000

type replicateDiffUI struct {
	spinner  spinner.Model
	sub      <-chan madmin.DiffInfo
	arn      string
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

func initReplicateDiffUI(arn string, diffCh <-chan madmin.DiffInfo) *replicateDiffUI {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	columns := getDiffHeader()

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	ts := getDiffStyles()
	t.SetStyles(ts)

	return &replicateDiffUI{
		spinner: s,
		sub:     diffCh,
		arn:     arn,
		table:   t,
		help:    help.New(),
		keymap:  newKeyMap(),
	}
}

func (m *replicateDiffUI) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		waitForActivity(m.sub), // wait for activity
	)
}

// A command that waits for the activity on a channel.
func waitForActivity(sub <-chan madmin.DiffInfo) tea.Cmd {
	return func() tea.Msg {
		msg := <-sub
		return msg
	}
}

func getDiffHeader() []table.Column {
	return []table.Column{
		{Title: "Attempted At", Width: 23},
		{Title: "Created", Width: 23},
		{Title: "Status", Width: 9},
		{Title: "VersionID", Width: 36},
		{Title: "Op", Width: 3},
		{Title: "Object", Width: 60},
	}
}

func getDiffStyles() table.Styles {
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

func (m *replicateDiffUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			columns := getDiffHeader()
			ts := getDiffStyles()
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
				rdif := replicateDiffMessage{
					DiffInfo: msg,
					arn:      m.arn,
				}
				m.rows = append(m.rows, rdif.toRow())
			}
			return m, waitForActivity(m.sub)
		}
		m.quitting = true
		columns := getDiffHeader()
		ts := getDiffStyles()
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

func (m *replicateDiffUI) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.enter,
		m.keymap.down,
		m.keymap.up,
		m.keymap.quit,
	})
}

func (m *replicateDiffUI) View() string {
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

func mainReplicateDiff(cliCtx *cli.Context) error {
	checkReplicateDiffSyntax(cliCtx)
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
	verbose := cliCtx.Bool("verbose")
	arn := cliCtx.String("arn")
	diffCh := client.BucketReplicationDiff(ctx, bucket, madmin.ReplDiffOpts{
		Verbose: verbose,
		ARN:     arn,
		Prefix:  prefix,
	})
	if globalJSON {
		for di := range diffCh {
			console.Println(replicateDiffMessage{
				Op:       "diff",
				DiffInfo: di,
				arn:      arn,
				verbose:  verbose,
			}.JSON())
		}
		return nil
	}

	ui := tea.NewProgram(initReplicateDiffUI(arn, diffCh))
	if _, e := ui.Run(); e != nil {
		cancel()
		fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to fetch replication diff")
	}
	return nil
}
