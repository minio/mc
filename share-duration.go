/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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
	"fmt"
	"time"
)

// shareDuration extended version of time.Duration implementing .Days() for convenience
type shareDuration struct {
	duration time.Duration
}

func (s shareDuration) Days() float64 {
	return s.duration.Hours() / 24
}

func (s shareDuration) Seconds() float64 {
	return s.duration.Seconds()
}

func (s shareDuration) Hours() float64 {
	return s.duration.Hours()
}

func (s shareDuration) String() string {
	if s.duration.Hours() > 24 {
		return fmt.Sprintf("%dd", int64(s.Days()))
	}
	return s.duration.String()
}

func (s shareDuration) GetDuration() time.Duration {
	return s.duration
}
