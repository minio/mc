package donut

import (
	"errors"

	"github.com/minio-io/mc/pkg/encoding/erasure"
)

type encoder struct {
	encoder   *erasure.Encoder
	k, m      uint8
	technique erasure.Technique
}

// getErasureTechnique - convert technique string into Technique type
func getErasureTechnique(technique string) (erasure.Technique, error) {
	switch true {
	case technique == "Cauchy":
		return erasure.Cauchy, nil
	case technique == "Vandermonde":
		return erasure.Cauchy, nil
	default:
		return erasure.None, errors.New("Invalid erasure technique")
	}
}

func NewEncoder(k, m uint8, technique string) (Encoder, error) {
	e := encoder{}
	t, err := getErasureTechnique(technique)
	if err != nil {
		return nil, err
	}
	params, err := erasure.ParseEncoderParams(k, m, t)
	if err != nil {
		return nil, err
	}
	e.encoder = erasure.NewEncoder(params)
	return e, nil
}
func (e encoder) Encode(data []byte) (encodedData [][]byte, err error) {
	if data == nil {
		return nil, errors.New("invalid argument")
	}
	encodedData, err = e.encoder.Encode(data)
	if err != nil {
		return nil, err
	}
	return encodedData, nil
}

func (e encoder) Decode(encodedData [][]byte, dataLength int) (data []byte, err error) {
	decodedData, err := e.encoder.Decode(encodedData, dataLength)
	if err != nil {
		return nil, err
	}
	return decodedData, nil
}
