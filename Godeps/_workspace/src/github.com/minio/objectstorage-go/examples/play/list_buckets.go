// +build ignore

/*
 * Minimal object storage library (C) 2015 Minio, Inc.
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
	"log"

	"github.com/minio/objectstorage-go"
)

func main() {
	config := new(objectstorage.Config)
	config.AccessKeyID = ""
	config.SecretAccessKey = ""
	config.Endpoint = "http://play.minio.io:9000"
	config.AcceptType = ""
	m := objectstorage.New(config)
	for message := range m.ListBuckets() {
		if message.Err != nil {
			log.Fatal(message.Err)
		}
		log.Println(message.Data)
	}
}
