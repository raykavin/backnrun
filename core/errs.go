package core

import "errors"

var (
	ErrBaseAssetEmpty  = errors.New("empty base asset")
	ErrQuoteAssetEmpty = errors.New("empty quote asset")
	ErrNegativeValue   = errors.New("negative value")
)
