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
	"strings"
	"testing"
)

func TestMakeCurlCmdEscapesSpecialChars(t *testing.T) {
	testCases := []struct {
		key         string
		expectedKey string
	}{
		{
			key:         "Robert O'Neil.png",
			expectedKey: "Robert\\ O\\'Neil.png",
		},
		{
			key:         "A&B-Design<Revision 1>|2014.pdf",
			expectedKey: "A\\&B-Design\\<Revision\\ 1\\>\\|2014.pdf",
		},
		{
			key:         "A&B-Design(Revision 1).pdf",
			expectedKey: "A\\&B-Design\\(Revision\\ 1\\).pdf",
		},
		{
			key:         "Matt`s\tResume.pdf",
			expectedKey: "Matt\\`s\\\tResume.pdf",
		},
		{
			key:         "out.pdf;rm -rf $HOME #",
			expectedKey: "out.pdf\\;rm\\ -rf\\ \\$HOME\\ \\#",
		},
	}

	for _, testCase := range testCases {
		cmd, _ := makeCurlCmd(testCase.key, "http://example.com", false, map[string]string{})
		if !strings.Contains(cmd, " key="+testCase.expectedKey+" ") {
			t.Errorf("Did not find key=%s in command %s", testCase.expectedKey, cmd)
		}
	}
}
