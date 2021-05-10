// Copyright (c) 2015-2021 MinIO, Inc.
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
	"os"
	"os/signal"
)

// trapSignals traps the registered signals and cancel the global context.
func trapSignals(sig ...os.Signal) {
	// channel to receive signals.
	sigCh := make(chan os.Signal, 1)
	defer close(sigCh)

	// `signal.Notify` registers the given channel to
	// receive notifications of the specified signals.
	signal.Notify(sigCh, sig...)

	// Wait for the signal.
	s := <-sigCh

	// Once signal has been received stop signal Notify handler.
	signal.Stop(sigCh)

	// Cancel the global context
	globalCancel()

	var exitCode int
	switch s.String() {
	case "interrupt":
		exitCode = globalCancelExitStatus
	case "killed":
		exitCode = globalKillExitStatus
	case "terminated":
		exitCode = globalTerminatExitStatus
	default:
		exitCode = globalErrorExitStatus
	}
	os.Exit(exitCode)
}
