/*
 * Minio Client (C) 2015 Minio, Inc.
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

package main

import (
	"bytes"
	"fmt"
	"strings"
)

// Check if version of the config is valid
func validateConfigVersion(config *configV8) (bool, string) {
	if config.Version != globalMCConfigVersion {
		return false, fmt.Sprintf("Config version %s does not match GlobalMCConfiguration %s.\n",
			config.Version, globalMCConfigVersion)
	}
	return true, ""
}

// Verifies the config file of the Minio Client
func validateConfigFile(config *configV8) (bool, []string) {
	ok, err := validateConfigVersion(config)
	var validationSuccessful = true
	var errors []string
	if !ok {
		validationSuccessful = false
		errors = append(errors, err)
	}
	hosts := config.Hosts
	for _, hostConfig := range hosts {
		hostConfigHealthOk, hostErrors := validateConfigHost(hostConfig)
		if !hostConfigHealthOk {
			validationSuccessful = false
			errors = append(errors, hostErrors...)
		}
	}
	return validationSuccessful, errors
}

func validateConfigHost(host hostConfigV8) (bool, []string) {
	var validationSuccessful = true
	var hostErrors []string
	api := host.API
	validAPI := isValidAPI(strings.ToLower(api))
	if !validAPI {
		var errorMsg bytes.Buffer
		errorMsg.WriteString(fmt.Sprintf(
			"%s API for host %s is not Valid. It is not part of any of the following APIs:\n",
			api, host.URL))
		for index, validAPI := range validAPIs {
			errorMsg.WriteString(fmt.Sprintf("%d. %s\n", index+1, validAPI))
		}
		validationSuccessful = false
		hostErrors = append(hostErrors, errorMsg.String())
	}
	url := host.URL
	validURL := isValidHostURL(url)
	if !validURL {
		validationSuccessful = false
		msg := fmt.Sprintf("URL %s for host %s is not valid. Could not parse it.\n", url, host.URL)
		hostErrors = append(hostErrors, msg)
	}
	return validationSuccessful, hostErrors
}
