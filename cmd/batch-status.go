package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
	"github.com/olekukonko/tablewriter"
)

var batchStatusCmd = cli.Command{
	Name:            "status",
	Usage:           "summarize job events on MinIO server in real-time",
	Action:          mainBatchStatus,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET JOBID

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. Display current in-progress JOB events.
      {{.Prompt}} {{.HelpName}} myminio/ KwSysDpxcBU9FNhGkn2dCf
`,
}

// batchJobStatusMessage container for batch job status messages
type batchJobStatusMessage struct {
	Status string           `json:"status"`
	Metric madmin.JobMetric `json:"metric"`
}

// JSON jsonified batchJobStatusMessage message
func (c batchJobStatusMessage) JSON() string {
	batchJobStatusMessageBytes, e := json.MarshalIndent(c, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(batchJobStatusMessageBytes)
}

func (c batchJobStatusMessage) String() string {
	return c.JSON()
}

// checkBatchStatusSyntax - validate all the passed arguments
func checkBatchStatusSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func mainBatchStatus(ctx *cli.Context) error {
	checkBatchStatusSyntax(ctx)

	aliasedURL := ctx.Args().Get(0)
	jobID := ctx.Args().Get(1)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	_, e := client.DescribeBatchJob(ctxt, jobID)
	nosuchJob := madmin.ToErrorResponse(e).Code == "XMinioAdminNoSuchJob"
	if nosuchJob {
		e = nil
		if !globalJSON {
			console.Infoln("Unable to find an active job, attempting to list from previously run jobs")
		}
	}
	fatalIf(probe.NewError(e), "Unable to lookup job status")

	ui := tea.NewProgram(initBatchJobMetricsUI(jobID))
	if nosuchJob {
		go func() {
			res, e := client.BatchJobStatus(ctxt, jobID)
			fatalIf(probe.NewError(e), "Unable to lookup job status")
			if globalJSON {
				printMsg(batchJobStatusMessage{
					Status: "success",
					Metric: res.LastMetric,
				})
				if res.LastMetric.Complete || res.LastMetric.Failed {
					cancel()
					return
				}
			} else {
				ui.Send(res.LastMetric)
			}
		}()
	} else {
		go func() {
			opts := madmin.MetricsOptions{
				Type:     madmin.MetricsBatchJobs,
				ByJobID:  jobID,
				Interval: time.Second,
			}
			e := client.Metrics(ctxt, opts, func(metrics madmin.RealtimeMetrics) {
				if globalJSON {
					if metrics.Aggregated.BatchJobs == nil {
						cancel()
						return
					}

					job, ok := metrics.Aggregated.BatchJobs.Jobs[jobID]
					if !ok {
						cancel()
						return
					}

					m := batchJobStatusMessage{
						Status: "in-progress",
						Metric: job,
					}
					switch {
					case job.Complete:
						m.Status = "complete"
					case job.Failed:
						m.Status = "failed"
					default:
						// leave as is with in-progress
					}
					printMsg(m)
					if job.Complete || job.Failed {
						cancel()
						return
					}
				} else {
					ui.Send(metrics.Aggregated.BatchJobs.Jobs[jobID])
				}
			})
			if e != nil && !errors.Is(e, context.Canceled) {
				fatalIf(probe.NewError(e).Trace(ctx.Args()...), "Unable to get current batch status")
			}
		}()
	}

	if !globalJSON {
		if _, e := ui.Run(); e != nil {
			cancel()
			fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to get current batch status")
		}
	} else {
		<-ctxt.Done()
	}

	return nil
}

func initBatchJobMetricsUI(jobID string) *batchJobMetricsUI {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return &batchJobMetricsUI{
		spinner: s,
		jobID:   jobID,
	}
}

type batchJobMetricsUI struct {
	metric   madmin.JobMetric
	spinner  spinner.Model
	quitting bool
	jobID    string
}

func (m *batchJobMetricsUI) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *batchJobMetricsUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}
	case madmin.JobMetric:
		m.metric = msg
		if msg.Complete || msg.Failed {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m *batchJobMetricsUI) View() string {
	var s strings.Builder

	// Set table header
	table := tablewriter.NewWriter(&s)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t") // pad with tabs
	table.SetNoWhiteSpace(true)

	var data [][]string
	addLine := func(prefix string, value any) {
		data = append(data, []string{
			prefix,
			whiteStyle.Render(fmt.Sprint(value)),
		})
	}

	if !m.quitting {
		s.WriteString(m.spinner.View())
	} else {
		if m.metric.Complete {
			s.WriteString(m.spinner.Style.Render((tickCell + tickCell + tickCell)))
		} else if m.metric.Failed {
			s.WriteString(m.spinner.Style.Render((crossTickCell + crossTickCell + crossTickCell)))
		}
	}
	s.WriteString("\n")

	switch m.metric.JobType {
	case string(madmin.BatchJobReplicate):
		accElapsedTime := m.metric.LastUpdate.Sub(m.metric.StartTime)

		addLine("JobType: ", m.metric.JobType)
		addLine("Objects: ", m.metric.Replicate.Objects)
		addLine("Versions: ", m.metric.Replicate.Objects)
		addLine("FailedObjects: ", m.metric.Replicate.ObjectsFailed)
		addLine("DeleteMarker: ", m.metric.Replicate.DeleteMarkers)
		addLine("FailedDeleteMarker: ", m.metric.Replicate.DeleteMarkersFailed)
		if accElapsedTime > 0 {
			bytesTransferredPerSec := float64(m.metric.Replicate.BytesTransferred) / accElapsedTime.Seconds()
			objectsPerSec := float64(int64(time.Second)*m.metric.Replicate.Objects) / float64(accElapsedTime)
			addLine("Throughput: ", fmt.Sprintf("%s/s", humanize.IBytes(uint64(bytesTransferredPerSec))))
			addLine("IOPs: ", fmt.Sprintf("%.2f objs/s", objectsPerSec))
		}
		addLine("Transferred: ", humanize.IBytes(uint64(m.metric.Replicate.BytesTransferred)))
		addLine("Elapsed: ", accElapsedTime.Round(time.Second).String())
		addLine("CurrObjName: ", m.metric.Replicate.Object)
	case string(madmin.BatchJobExpire):
		addLine("JobType: ", m.metric.JobType)
		addLine("Objects: ", m.metric.Expired.Objects)
		addLine("FailedObjects: ", m.metric.Expired.ObjectsFailed)
		addLine("DeleteMarker: ", m.metric.Expired.DeleteMarkers)
		addLine("FailedDeleteMarker: ", m.metric.Expired.DeleteMarkersFailed)
		addLine("CurrObjName: ", m.metric.Expired.Object)

		if !m.metric.LastUpdate.IsZero() {
			accElapsedTime := m.metric.LastUpdate.Sub(m.metric.StartTime)
			addLine("Elapsed: ", accElapsedTime.String())
		}
	case string(madmin.BatchJobCatalog):
		addLine("JobType: ", m.metric.JobType)
		addLine("ObjectsScannedCount: ", m.metric.Catalog.ObjectsScannedCount)
		addLine("ObjectsMatchedCount: ", m.metric.Catalog.ObjectsMatchedCount)
		lastScanned := fmt.Sprintf("%s/%s", m.metric.Catalog.LastBucketScanned, m.metric.Catalog.LastObjectScanned)
		addLine("LastScanned: ", lastScanned)
		lastMatched := fmt.Sprintf("%s/%s", m.metric.Catalog.LastBucketMatched, m.metric.Catalog.LastObjectMatched)
		addLine("LastMatched: ", lastMatched)
		accElapsedTime := m.metric.LastUpdate.Sub(m.metric.StartTime)
		addLine("RecordsWrittenCount: ", m.metric.Catalog.RecordsWrittenCount)
		addLine("OutputObjectsCount: ", m.metric.Catalog.OutputObjectsCount)
		addLine("Elapsed: ", accElapsedTime.Round(time.Second).String())
		addLine("Scan Speed: ", fmt.Sprintf("%f objects/s", float64(m.metric.Catalog.ObjectsScannedCount)/accElapsedTime.Seconds()))
		if m.metric.Catalog.ErrorMsg != "" {
			addLine("Error: ", m.metric.Catalog.ErrorMsg)
		}
	}

	table.AppendBulk(data)
	table.Render()

	if m.quitting {
		s.WriteString("\n")
	}
	return s.String()
}
