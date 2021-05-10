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
	"fmt"
	"strings"
)

// Check if version of the config is valid
func validateConfigVersion(config *configV10) (bool, string) {
	if config.Version != globalMCConfigVersion {
		return false, fmt.Sprintf("Config version '%s' does not match mc config version '%s', please update your binary.\n",
			config.Version, globalMCConfigVersion)
	}
	return true, ""
}

// Verifies the config file of the MinIO Client
func validateConfigFile(config *configV10) (bool, []string) {
	ok, err := validateConfigVersion(config)
	var validationSuccessful = true
	var errors []string
	if !ok {
		validationSuccessful = false
		errors = append(errors, err)
	}
	aliases := config.Aliases
	for _, aliasConfig := range aliases {
		aliasConfigHealthOk, aliasErrors := validateConfigHost(aliasConfig)
		if !aliasConfigHealthOk {
			validationSuccessful = false
			errors = append(errors, aliasErrors...)
		}
	}
	return validationSuccessful, errors
}

func validateConfigHost(host aliasConfigV10) (bool, []string) {
	var validationSuccessful = true
	var hostErrors []string
	if !isValidAPI(strings.ToLower(host.API)) {
		validationSuccessful = false
		hostErrors = append(hostErrors, errInvalidAPISignature(host.API, host.URL).ToGoError().Error())
	}
	if !isValidHostURL(host.URL) {
		validationSuccessful = false
		hostErrors = append(hostErrors, errInvalidURL(host.URL).ToGoError().Error())
	}
	return validationSuccessful, hostErrors
}
