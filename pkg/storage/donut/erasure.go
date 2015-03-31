package donut

import (
	"errors"

	encoding "github.com/minio-io/donut/erasure"
)

type encoder struct {
	encoder   *encoding.Erasure
	k, m      uint8
	technique encoding.Technique
}

// getErasureTechnique - convert technique string into Technique type
func getErasureTechnique(technique string) (encoding.Technique, error) {
	switch true {
	case technique == "Cauchy":
		return encoding.Cauchy, nil
	case technique == "Vandermonde":
		return encoding.Cauchy, nil
	default:
		return encoding.None, errors.New("Invalid erasure technique")
	}
}

// NewEncoder - instantiate a new encoder
func NewEncoder(k, m uint8, technique string) (Encoder, error) {
	e := encoder{}
	t, err := getErasureTechnique(technique)
	if err != nil {
		return nil, err
	}
	params, err := encoding.ValidateParams(k, m, t)
	if err != nil {
		return nil, err
	}
	e.encoder = encoding.NewErasure(params)
	e.k = k
	e.m = m
	e.technique = t
	return e, nil
}

func (e encoder) GetEncodedBlockLen(dataLength int) (int, error) {
	if dataLength == 0 {
		return 0, errors.New("invalid argument")
	}
	return encoding.GetEncodedBlockLen(dataLength, e.k), nil
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
