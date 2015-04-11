/*
 * QConfig - Quick way to implement a configuration file
 *
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
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

package qconfig

import (
	"testing"

	. "github.com/minio-io/check"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func (s *MySuite) TestVersion(c *C) {
	cfg := NewConfig(Version{1, 0, 0})
	c.Assert(cfg.GetVersion(), DeepEquals, Version{1, 0, 0})

	c.Assert(Str2Version("1.0.0"), DeepEquals, cfg.GetVersion())
}

func (s *MySuite) TestSaveLoad(c *C) {
	version := Version{1, 0, 0}

	cfg := NewConfig(version)
	cfg.SetFloat64("Pi", 3.1415)
	cfg.SaveConfig("test.json")

	newCfg := NewConfig(version)
	newCfg.LoadConfig("test.json")
	pi := newCfg.GetFloat64("Pi")
	c.Assert(pi, Equals, 3.1415)
}

func (s *MySuite) TestGetSet(c *C) {
	version := Version{1, 0, 0}
	cfg := NewConfig(version)

	cfg.SetInt("Q", 42)
	c.Assert(cfg.GetInt("Q"), Equals, 42)

	cfg.SetIntSlice("Odd", []int{1, 3, 5, 7, 9})
	c.Assert(cfg.GetIntSlice("Odd"), DeepEquals, []int{1, 3, 5, 7, 9})

	cfg.SetFloat64("Pi", 3.1415)
	c.Assert(cfg.GetFloat64("Pi"), Equals, 3.1415)

	cfg.SetFloat64Slice("Pi", []float64{3.1415, 2.414})
	c.Assert(cfg.GetFloat64Slice("Pi"), DeepEquals, []float64{3.1415, 2.414})

	cfg.SetString("Grand Nagus", "Zek")
	c.Assert(cfg.GetString("Grand Nagus"), Equals, "Zek")

	cfg.SetStringSlice("Ferengi", []string{"Zek", "Brunt", "Quark", "Rom", "Nog", "Ishka"})
	c.Assert(cfg.GetStringSlice("Ferengi"), DeepEquals, []string{"Zek", "Brunt", "Quark", "Rom", "Nog", "Ishka"})

	startrek1 := map[string]string{"Borg": "7of9", "Data": "Measure of a Man"}
	startrek2 := map[string]string{"Borg": "7of9", "Data": "Measure of a Man"}
	cfg.SetMapString("startrek", startrek1)
	c.Assert(cfg.GetMapString("startrek"), DeepEquals, startrek2)

	startrek3 := map[string][]string{
		"Quadrants": []string{"Alpha", "Beta", "Gamma", "Delta"},
		"Aliens":    []string{"Dominion", "Borg", "Klingon", "Romulan"},
	}
	startrek4 := map[string][]string{
		"Quadrants": []string{"Alpha", "Beta", "Gamma", "Delta"},
		"Aliens":    []string{"Dominion", "Borg", "Klingon", "Romulan"},
	}
	cfg.SetMapStringSlice("startrek", startrek3)
	c.Assert(cfg.GetMapStringSlice("startrek"), DeepEquals, startrek4)

	startrek5 := map[string][]string{
		"Quadrants": []string{"Beta", "Gamma", "Delta"},
		"Aliens":    []string{"Dominion", "Borg", "Klingon", "Romulan"},
	}
	startrek6 := map[string][]string{
		"Quadrants": []string{"Alpha", "Beta", "Gamma", "Delta"},
		"Aliens":    []string{"Dominion", "Borg", "Klingon", "Romulan"},
	}
	cfg.SetMapStringSlice("startrek", startrek5)
	c.Assert(cfg.GetMapStringSlice("startrek"), Not(DeepEquals), startrek6)

	c.Assert(cfg.GetMapStringSlice("Startrek"), IsNil)

}
