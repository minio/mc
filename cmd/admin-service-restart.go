// Copyright (c) 2015-2024 MinIO, Inc.
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
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var serviceRestartFlag = []cli.Flag{
	cli.BoolFlag{
		Name:  "dry-run",
		Usage: "do not attempt a restart, however verify the peer status",
	},
	cli.BoolFlag{
		Name:  "wait, w",
		Usage: "wait for background initializations to complete",
	},
}

var adminServiceRestartCmd = cli.Command{
	Name:         "restart",
	Usage:        "restart a MinIO cluster",
	Action:       mainAdminServiceRestart,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(serviceRestartFlag, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Restart MinIO server represented by its alias 'play'.
     {{.Prompt}} {{.HelpName}} play/
`,
}

type serviceRestartUI struct {
	current  atomic.Value
	tbl      *console.Table
	meter    spinner.Model
	quitting bool
}

func (m *serviceRestartUI) add(msg serviceRestartMessage) {
	m.current.Store(msg)
}

func (m *serviceRestartUI) Init() tea.Cmd {
	return m.meter.Tick
}

func (m *serviceRestartUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.quitting {
		return m, tea.Quit
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.meter, cmd = m.meter.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *serviceRestartUI) View() string {
	var s strings.Builder

	msgI := m.current.Load()
	if msgI == nil {
		s.WriteString("(waiting for data)")
		return s.String()
	}

	s.WriteString(fmt.Sprintf("Service status: %s ", m.meter.View()))

	msg := msgI.(serviceRestartMessage)
	state := msg.State
	switch state {
	case restarting:
		// Actually still restarting, no response yet from the server.
		s.WriteString("[RESTARTING]\n")
	case waiting:
		// Waiting on background initializations such as IAM and bucket metadata
		s.WriteString(console.Colorize("ServiceInitializing", "[WAITING]"))
		s.WriteString("\n")
	case done:
		m.quitting = true

		// Finished restarting and optionally waiting for the background initializations to complete.
		s.WriteString(console.Colorize("ServiceRestarted", "[DONE]"))
		s.WriteString("\n")
		var (
			totalNodes        = len(msg.Result.Results)
			totalOfflineNodes int
			totalHungNodes    int
		)

		for _, peerRes := range msg.Result.Results {
			if peerRes.Err != "" {
				totalOfflineNodes++
			} else if len(peerRes.WaitingDrives) > 0 {
				totalHungNodes++
			}
		}

		s.WriteString("Summary:\n")

		var cellText [][]string
		cellText = append(cellText, []string{
			"Servers:",
			fmt.Sprintf("%d online, %d offline, %d hung", totalNodes-(totalOfflineNodes+totalHungNodes), totalOfflineNodes, totalHungNodes),
		})

		cellText = append(cellText, []string{
			"Restart Time:",
			msg.RestartDuration.String(),
		})

		if msg.WaitingDuration > 0 {
			cellText = append(cellText, []string{
				"Background Init Time:",
				msg.WaitingDuration.String(),
			})
		}

		fatalIf(probe.NewError(m.tbl.PopulateTable(&s, cellText)), "unable to populate the table")
	}

	return s.String()
}

func initServiceRestartUI(rowCount int, currentCh chan serviceRestartMessage) *serviceRestartUI {
	var printColors []*color.Color
	for range rowCount {
		printColors = append(printColors, getPrintCol(colGreen))
	}

	tbl := console.NewTable(printColors, []bool{false, false}, 4)

	meter := spinner.New()
	meter.Spinner = spinner.Meter
	meter.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	svcUI := &serviceRestartUI{tbl: tbl, meter: meter}
	go func() {
		for msg := range currentCh {
			if msg.Status != "" {
				svcUI.add(msg)
			}
		}
	}()
	return svcUI
}

const (
	restarting = iota
	waiting
	done
)

// serviceRestartMessage is container for service restart command success and failure messages.
type serviceRestartMessage struct {
	Status          string                     `json:"status"`
	ServerURL       string                     `json:"serverURL"`
	Result          madmin.ServiceActionResult `json:"result"`
	RestartDuration time.Duration              `json:"restartDuration"`
	WaitingDuration time.Duration              `json:"waitingDuration"`
	TimeTaken       time.Duration              `json:"timeTaken"` // deprecated use "restartDuration" instead.
	State           int                        `json:"state"`
}

func (s serviceRestartMessage) String() string {
	return s.JSON()
}

// JSON jsonified service restart command message.
func (s serviceRestartMessage) JSON() string {
	serviceRestartJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(serviceRestartJSONBytes)
}

// checkAdminServiceRestartSyntax - validate all the passed arguments
func checkAdminServiceRestartSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func mainAdminServiceRestart(ctx *cli.Context) error {
	// Validate serivce restart syntax.
	checkAdminServiceRestartSyntax(ctx)

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	// Set color.
	console.SetColor("ServiceOffline", color.New(color.FgRed, color.Bold))
	console.SetColor("ServiceInitializing", color.New(color.FgYellow, color.Bold))
	console.SetColor("ServiceRestarted", color.New(color.FgGreen, color.Bold))
	console.SetColor("FailedServiceRestart", color.New(color.FgRed, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	rowCount := 2
	toWait := ctx.Bool("wait")
	if toWait {
		rowCount = 3
	}

	ch := make(chan serviceRestartMessage, 1)

	svcUI := initServiceRestartUI(rowCount, ch)
	go func() {
		t := time.Now()

		// Restart the specified MinIO server
		result, e := client.ServiceAction(ctxt, madmin.ServiceActionOpts{
			Action: madmin.ServiceActionRestart,
			DryRun: ctx.Bool("dry-run"),
		})
		if e != nil {
			// Attempt an older API server might be old
			// nolint:staticcheck
			// we need this fallback
			e = client.ServiceRestart(ctxt)
		}
		fatalIf(probe.NewError(e), "Unable to restart the server.")

		timeTaken := time.Since(t)
		restart := restarting
		if !toWait {
			restart = done
		}

		ch <- serviceRestartMessage{
			Status:          "success",
			ServerURL:       aliasedURL,
			Result:          result,
			RestartDuration: timeTaken,
			TimeTaken:       timeTaken,
			State:           restart,
		}

		if toWait {
			sleepInterval := 500 * time.Millisecond
			go func() {
				defer close(ch)

				wt := time.Now()

				// Start pinging the service until it is ready
				anonClient, err := newAnonymousClient(aliasedURL)
				fatalIf(err.Trace(aliasedURL), "unable to initialize anonymous client for`"+aliasedURL+"`.")

				for {
					healthCtx, healthCancel := context.WithTimeout(ctxt, 2*time.Second)

					// Fetch the health status of the specified MinIO server
					healthResult, healthErr := anonClient.Healthy(healthCtx, madmin.HealthOpts{})
					healthCancel()

					switch {
					case healthErr == nil && healthResult.Healthy:
						ch <- serviceRestartMessage{
							Status:          "success",
							ServerURL:       aliasedURL,
							Result:          result,
							RestartDuration: timeTaken,
							TimeTaken:       timeTaken,
							WaitingDuration: time.Since(wt),
							State:           done,
						}
						return
					}

					ch <- serviceRestartMessage{
						Status:          "success",
						ServerURL:       aliasedURL,
						Result:          result,
						WaitingDuration: time.Since(wt),
						RestartDuration: timeTaken,
						TimeTaken:       timeTaken,
						State:           waiting,
					}

					time.Sleep(sleepInterval)
				}
			}()
		} else {
			close(ch)
		}
	}()

	if !globalJSON {
		ui := tea.NewProgram(svcUI)
		if _, e := ui.Run(); e != nil {
			cancel()
			fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to initialize service restart UI")
		}
	} else {
		for msg := range ch {
			printMsg(msg)
			if msg.State == done {
				break
			}
		}
	}

	return nil
}
