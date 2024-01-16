// Copyright (c) 2015-2022 MinIO, Inc.
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
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

var percentStyle = lipgloss.NewStyle().Width(4).Align(lipgloss.Left)

type model struct {
	viewport viewport.Model
	content  string

	ready        bool
	renderedOnce chan struct{}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case string:
		m.content += msg
		m.viewport.SetContent(wordwrap.String(m.content, m.viewport.Width-2))
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.content)
			m.ready = true
			close(m.renderedOnce)
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	}

	// Handle keyboard and mouse events in the viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m model) headerView() string {
	info := " (q)uit/esc"
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (m model) footerView() string {
	// Disables printing if the viewport is not ready
	if m.viewport.Width == 0 {
		return ""
	}
	if math.IsNaN(m.viewport.ScrollPercent()) {
		return ""
	}

	viewP := int(m.viewport.ScrollPercent() * 100)
	info := fmt.Sprintf(" %s", percentStyle.Render(fmt.Sprintf("%d%%", viewP)))
	totalLength := m.viewport.Width - lipgloss.Width(info)
	finishedCount := int((float64(totalLength) / 100) * float64(viewP))

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		info,
		strings.Repeat("/", finishedCount),
		strings.Repeat("─", max(0, totalLength-finishedCount)),
	)
}

type termPager struct {
	initialized bool

	model    *model
	teaPager *tea.Program

	buf      chan []byte
	statusCh chan error
}

func (tp *termPager) init() {
	tp.statusCh = make(chan error)
	tp.buf = make(chan []byte)
	tp.model = &model{renderedOnce: make(chan struct{})}
	go func() {
		tp.teaPager = tea.NewProgram(
			tp.model,
		)

		go func() {
			_, e := tp.teaPager.Run()
			tp.statusCh <- e
			close(tp.statusCh)
		}()

		fallback := false
		select {
		case <-tp.model.renderedOnce:
		case err := <-tp.statusCh:
			if err != nil {
				fallback = true
			}
		}
		for {
			select {
			case s := <-tp.buf:
				if !fallback {
					tp.teaPager.Send(string(s))
				} else {
					os.Stdout.Write(s)
				}
			case <-tp.statusCh:
				return
			}
		}
	}()
	tp.initialized = true
}

func (tp *termPager) Write(p []byte) (int, error) {
	if !tp.initialized {
		tp.init()
	}
	tp.buf <- p
	return len(p), nil
}

func (tp *termPager) WaitForExit() {
	if !tp.initialized {
		return
	}
	// Wait until the term pager this is closed
	// which is trigerred when there is an error
	// or the user quits
	for status := range tp.statusCh {
		_ = status
	}
}

func newTermPager() *termPager {
	return &termPager{}
}
