// Copyright (c) 2015-2024 MinIO, Inc.
//
// # This file is part of MinIO Object Storage stack
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
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

type sseKeyType int

const (
	sseNone sseKeyType = iota
	sseC
	sseKMS
	sseS3
)

// struct representing object prefix and sse keys association.
type prefixSSEPair struct {
	Prefix string
	SSE    encrypt.ServerSide
}

// byPrefixLength implements sort.Interface.
type byPrefixLength []prefixSSEPair

func (p byPrefixLength) Len() int { return len(p) }
func (p byPrefixLength) Less(i, j int) bool {
	if len(p[i].Prefix) != len(p[j].Prefix) {
		return len(p[i].Prefix) > len(p[j].Prefix)
	}
	return p[i].Prefix < p[j].Prefix
}

func (p byPrefixLength) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// get SSE Key if object prefix matches with given resource.
func getSSE(resource string, encKeys []prefixSSEPair) encrypt.ServerSide {
	resource = filepath.ToSlash(resource)
	for _, k := range encKeys {
		if strings.HasPrefix(resource, k.Prefix) {
			return k.SSE
		}
	}
	return nil
}

func validateAndCreateEncryptionKeys(ctx *cli.Context) (encMap map[string][]prefixSSEPair, err *probe.Error) {
	encMap = make(map[string][]prefixSSEPair, 0)

	for _, v := range ctx.StringSlice("enc-kms") {
		prefixPair, alias, err := validateAndParseKey(ctx, v, sseKMS)
		if err != nil {
			return nil, err
		}
		encMap[alias] = append(encMap[alias], *prefixPair)
	}

	for _, v := range ctx.StringSlice("enc-s3") {
		prefixPair, alias, err := validateAndParseKey(ctx, v, sseS3)
		if err != nil {
			return nil, err
		}
		encMap[alias] = append(encMap[alias], *prefixPair)
	}

	for _, v := range ctx.StringSlice("enc-c") {
		prefixPair, alias, err := validateAndParseKey(ctx, v, sseC)
		if err != nil {
			return nil, err
		}
		encMap[alias] = append(encMap[alias], *prefixPair)
	}

	for i := range encMap {
		err = validateOverLappingSSEKeys(encMap[i])
		if err != nil {
			return nil, err
		}
	}

	for alias, ps := range encMap {
		if hostCfg := mustGetHostConfig(alias); hostCfg == nil {
			for _, p := range ps {
				return nil, errSSEInvalidAlias(p.Prefix)
			}
		}
	}

	for _, encKeys := range encMap {
		sort.Sort(byPrefixLength(encKeys))
	}

	return encMap, nil
}

func validateAndParseKey(ctx *cli.Context, key string, keyType sseKeyType) (SSEPair *prefixSSEPair, alias string, perr *probe.Error) {
	matchedCount := 0
	alias, prefix, encKey, keyErr := parseSSEKey(key, keyType)
	if keyErr != nil {
		return nil, "", keyErr
	}
	if alias == "" {
		return nil, "", errSSEInvalidAlias(prefix).Trace(key)
	}

	if (keyType == sseKMS || keyType == sseC) && encKey == "" {
		return nil, "", errSSEClientKeyFormat("SSE-C/KMS key should be of the form alias/prefix=key,... ").Trace(key)
	}

	ssePairPrefix := alias + "/" + prefix

	for _, arg := range ctx.Args() {
		if strings.HasPrefix(arg, ssePairPrefix) {
			matchedCount++
		} else if strings.HasPrefix(ssePairPrefix, arg) {
			matchedCount++
		}
	}

	if matchedCount == 0 {
		return nil, "", errSSEPrefixMatch()
	}

	var sse encrypt.ServerSide
	var err error

	switch keyType {
	case sseC:
		sse, err = encrypt.NewSSEC([]byte(encKey))
	case sseKMS:
		sse, err = encrypt.NewSSEKMS(encKey, nil)
	case sseS3:
		sse = encrypt.NewSSE()
	}

	if err != nil {
		return nil, "", probe.NewError(err).Trace(key)
	}

	return &prefixSSEPair{
		Prefix: ssePairPrefix,
		SSE:    sse,
	}, alias, nil
}

func validateOverLappingSSEKeys(keyMap []prefixSSEPair) (err *probe.Error) {
	for i := range keyMap {
		for j := i + 1; j < len(keyMap); j++ {
			if strings.HasPrefix(keyMap[i].Prefix, keyMap[j].Prefix) ||
				strings.HasPrefix(keyMap[j].Prefix, keyMap[i].Prefix) {
				return errSSEOverlappingAlias(keyMap[i].Prefix, keyMap[j].Prefix)
			}
		}
	}
	return
}

func splitKey(sseKey string) (alias, prefix string) {
	x := strings.SplitN(sseKey, "/", 2)
	switch len(x) {
	case 2:
		return x[0], x[1]
	case 1:
		return x[0], ""
	}
	return "", ""
}

func parseSSEKey(sseKey string, keyType sseKeyType) (
	alias string,
	prefix string,
	key string,
	err *probe.Error,
) {
	sseKeyBytes := []byte(sseKey)

	separatorIndex := bytes.LastIndex(sseKeyBytes, []byte("="))
	if separatorIndex < 0 {
		if keyType == sseS3 {
			alias, prefix = splitKey(sseKey)
			return
		}
		err = errSSEKeyMissing().Trace(sseKey)
		return
	}
	if separatorIndex == len(sseKeyBytes)-1 {
		err = errSSEKeyMissing().Trace(sseKey)
		return
	}

	encodedKey := string(sseKeyBytes[separatorIndex+1:])
	alias, prefix = splitKey(string(sseKeyBytes[:separatorIndex]))
	if keyType == sseKMS {
		if !validKMSKeyName(encodedKey) {
			err = errSSEKMSKeyFormat(fmt.Sprintf("Key (%s) is badly formatted.", encodedKey)).Trace(sseKey)
		}
		key = encodedKey
		return
	}

	var keyB []byte
	var decodeError error
	if len(encodedKey) == 64 {
		keyB, decodeError = hex.DecodeString(encodedKey)
	} else {
		keyB, decodeError = base64.RawStdEncoding.DecodeString(encodedKey)
	}

	if decodeError != nil {
		err = errSSEClientKeyFormat(fmt.Sprintf("Key (%s) was neither base64 raw encoded nor hex encoded.", encodedKey)).Trace(sseKey)
	}

	key = string(keyB)
	if len(key) != 32 {
		err = errSSEClientKeyFormat(fmt.Sprintf("Plain text from key (%s) is only %d bytes, but should be 32 bytes.", encodedKey, len(key))).Trace(sseKey)
		return
	}

	return
}

func validKMSKeyName(s string) bool {
	if s == "" || s == "_" {
		return false
	}

	n := len(s) - 1
	for i, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r == '-' && i > 0 && i < n:
		case r == '_':
		default:
			return false
		}
	}
	return true
}
