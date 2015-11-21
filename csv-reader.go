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

// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Some of the code here is borrowed from 'encoding/json' package.
package main

import (
	"encoding/csv"
	"io"
	"reflect"
	"runtime"
	"strconv"

	"github.com/minio/minio-xl/pkg/probe"
)

// csvReader reads records from a CSV-encoded file.
type csvReader struct {
	*csv.Reader
}

// newCSVReader returns a new csv Reader.
// additionally one can provide a line that needs to be skipped.
func newCSVReader(r io.Reader) *csvReader {
	return &csvReader{
		csv.NewReader(r),
	}
}

// An invalidUnmarshalError describes an invalid argument passed to Unmarshal.
// (The argument to Unmarshal must be a non-nil pointer.)
type invalidUnmarshalError struct {
	Type reflect.Type
}

func (e *invalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "csv: Unmarshal(nil)"
	}
	if e.Type.Kind() != reflect.Ptr {
		return "csv: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "csv: Unmarshal(nil " + e.Type.String() + ")"
}

// An unmarshalTypeError describes a CSV value that was
// not appropriate for a value of a specific Go type.
type unmarshalTypeError struct {
	Type reflect.Type // type of Go value it could not be assigned to
}

func (e *unmarshalTypeError) Error() string {
	return "csv: cannot unmarshal into Go value of type " + e.Type.String()
}

// Unmarshal - unmarshal everything into the input struct.
func (r *csvReader) Unmarshal(v interface{}, skipLine int) (err *probe.Error) {
	// Handler for runtime errors.
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			e := r.(error)
			err = probe.NewError(e)
		}
	}()
	{
		// Check if incoming interface{} is nil.
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Ptr || rv.IsNil() {
			return probe.NewError(&invalidUnmarshalError{reflect.TypeOf(v)})
		}
	}
	rv := reflect.ValueOf(v).Elem()
	rt := reflect.TypeOf(v).Elem().Elem()
	records := reflect.MakeSlice(reflect.TypeOf(v).Elem(), 0, 0)
	var e error
	var lineCount int
	for {
		record := reflect.New(rt)
		{
			var csvRecord []string
			csvRecord, e = r.Read()
			if e != nil {
				break
			}
			lineCount++
			// Skip a particular line
			if lineCount == skipLine {
				continue
			}
			rvInternal := reflect.ValueOf(record.Interface()).Elem()
			for s := 0; s < rvInternal.NumField(); s++ {
				val := rvInternal.Field(s)
				x := csvRecord[s]
				setValue(&val, x)
			}
		}
		records = reflect.Append(records, record.Elem())
	}
	if e == io.EOF {
		rv.Set(records)
		return nil
	}
	if e != nil {
		return probe.NewError(e)
	}
	return nil
}

// setValue - following reflect.Value sets the actual value based on the Golang type.
// If no native type found throws an UnmarshalTypeError.
func setValue(v *reflect.Value, x string) *probe.Error {
	switch v.Kind() {
	case reflect.Bool:
		val, e := strconv.ParseBool(x)
		if e != nil {
			return probe.NewError(e)
		}
		v.SetBool(val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, e := strconv.ParseInt(x, 10, v.Type().Bits())
		if e != nil {
			return probe.NewError(e)
		}
		v.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		val, e := strconv.ParseUint(x, 10, v.Type().Bits())
		if e != nil {
			return probe.NewError(e)
		}
		v.SetUint(val)

	case reflect.Float32, reflect.Float64:
		val, e := strconv.ParseFloat(x, v.Type().Bits())
		if e != nil {
			return probe.NewError(e)
		}
		v.SetFloat(val)

	case reflect.String:
		v.SetString(x)
	case reflect.Struct:
	case reflect.Map:
	case reflect.Slice:
	case reflect.Array:
	case reflect.Interface, reflect.Ptr:
	default:
		return probe.NewError(&unmarshalTypeError{v.Type()})
	}
	return nil
}
