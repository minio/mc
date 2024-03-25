package cmd

import (
	"encoding/base64"
	"sort"
	"strconv"
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
	return len(p[i].Prefix) > len(p[j].Prefix)
}
func (p byPrefixLength) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// get SSE Key if object prefix matches with given resource.
func getSSE(resource string, encKeys []prefixSSEPair) encrypt.ServerSide {
	for _, k := range encKeys {
		if strings.HasPrefix(resource, k.Prefix) {
			return k.SSE
		}
	}
	return nil
}

func validateAndGetEncryptionFlags(cliCtx *cli.Context) (
	sseKeys []string,
	sseType sseKeyType,
	err *probe.Error,
) {
	encKeyL := len(cliCtx.StringSlice("enc-c"))
	encS3L := len(cliCtx.StringSlice("enc-s3"))
	encKmsL := len(cliCtx.StringSlice("enc-kms"))

	if encKeyL > 1 && encS3L > 1 {
		err = errSSEParameterOverlap("--enc-s3", "--enc-c")
		return
	}
	if encKeyL > 1 && encKmsL > 1 {
		err = errSSEParameterOverlap("--enc-c", "--enc-kms")
		return
	}
	if encKmsL > 1 && encS3L > 1 {
		err = errSSEParameterOverlap("--enc-kms", "--enc-s3")
		return
	}

	if encKeyL > 0 {
		sseKeys = cliCtx.StringSlice("enc-c")
		sseType = sseC
	}
	if encS3L > 0 {
		sseKeys = cliCtx.StringSlice("enc-s3")
		sseType = sseS3
	}
	if encKmsL > 0 {
		sseKeys = cliCtx.StringSlice("enc-kms")
		sseType = sseKMS
	}

	return
}

// parse and return encryption key pairs per alias.
func validateAndCreateEncryptionKeys(ctx *cli.Context) (map[string][]prefixSSEPair, *probe.Error) {
	sseKeys, sseType, keyErr := validateAndGetEncryptionFlags(ctx)
	if keyErr != nil {
		return nil, keyErr
	}

	if sseType == sseNone {
		return nil, nil
	}

	return parseSSEKeys(ctx, sseKeys, sseType)
}

func validateOverLappingSSEKeys(keyMap []prefixSSEPair) (err *probe.Error) {
	for i := 0; i < len(keyMap); i++ {
		for j := i + 1; j < len(keyMap); j++ {
			if strings.HasPrefix(keyMap[i].Prefix, keyMap[j].Prefix) ||
				strings.HasPrefix(keyMap[j].Prefix, keyMap[i].Prefix) {
				return errSSEOverlappingAlias(keyMap[i].Prefix, keyMap[j].Prefix)
			}
		}
	}
	return
}

func parseSSEKeys(ctx *cli.Context, sseKeys []string, keyType sseKeyType) (encMap map[string][]prefixSSEPair, err *probe.Error) {
	encMap = make(map[string][]prefixSSEPair)
	matchedCount := 0

	for _, prefixAndKey := range sseKeys {

		alias, prefix, encKey, keyErr := parseSSEKey(prefixAndKey, keyType)
		if keyErr != nil {
			return nil, keyErr
		}

		if alias == "" {
			return encMap, errSSEInvalidAlias(prefix).Trace(prefixAndKey)
		}

		if (keyType == sseKMS || keyType == sseC) && encKey == "" {
			return encMap, errSSEClientKeyFormat("SSE-C/KMS key should be of the form alis/prefix=key,... ").Trace(prefixAndKey)
		}

		for _, arg := range ctx.Args() {
			if strings.HasPrefix(arg, alias+"/"+prefix) {
				matchedCount++
			}
		}

		if matchedCount == 0 {
			return nil, errSSEPrefixMatch()
		}

		var sse encrypt.ServerSide
		var err error
		var ssePairPrefix string

		switch keyType {
		case sseC:
			ssePairPrefix = alias + "/" + prefix
			sse, err = encrypt.NewSSEC([]byte(encKey))
			if err != nil {
				return encMap, probe.NewError(err).Trace(prefixAndKey)
			}
		case sseKMS:
			ssePairPrefix = alias + "/" + prefix
			sse, err = encrypt.NewSSEKMS(encKey, nil)
			if err != nil {
				return encMap, probe.NewError(err).Trace(prefixAndKey)
			}
		case sseS3:
			ssePairPrefix = alias + "/" + prefix
			sse = encrypt.NewSSE()
		}

		encMap[alias] = append(encMap[alias], prefixSSEPair{
			Prefix: ssePairPrefix,
			SSE:    sse,
		})

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

	return
}

func splitKey(sseKey string) (alias, prefix string) {
	x := strings.Split(sseKey, "/")
	if len(x) > 2 {
		return x[0], strings.Join(x[1:], "/")
	} else if len(x) == 2 {
		return x[0], x[1]
	}
	return x[0], ""
}

func parseSSEKey(sseKey string, keyType sseKeyType) (
	alias string,
	prefix string,
	key string,
	err *probe.Error,
) {
	if keyType == sseS3 {
		alias, prefix = splitKey(sseKey)
		return
	}

	var path string
	alias, path = splitKey(sseKey)
	splitPath := strings.Split(path, "=")
	if len(splitPath) == 0 {
		err = errSSEKeyMissing().Trace(sseKey)
		return
	}

	aliasPlusPrefix := strings.Join(splitPath[:len(splitPath)-1], "=")
	prefix = strings.Replace(aliasPlusPrefix, alias+"/", "", 1)
	key = splitPath[len(splitPath)-1]

	if keyType == sseC {
		keyB, de := base64.RawStdEncoding.DecodeString(key)
		if de != nil {
			err = errSSEClientKeyFormat("One of the inserted keys was " + strconv.Itoa(len(key)) + " bytes and did not have valid base64 raw encoding.").Trace(sseKey)
			return
		}
		key = string(keyB)
		if len(key) != 32 {
			err = errSSEClientKeyFormat("The plain text key was " + strconv.Itoa(len(key)) + " bytes but should be 32 bytes long").Trace(sseKey)
			return
		}
	}

	if keyType == sseKMS {
		if !validKMSKeyName(key) {
			err = errSSEKMSKeyFormat("One of the inserted keys was " + strconv.Itoa(len(key)) + " bytes and did not have a valid KMS key name.").Trace(sseKey)
			return
		}
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
