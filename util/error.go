package util

import "errors"

const (
	HTTP_OK_RESPONSE_CODE        = 0
	HTTP_READ_BODY_ERROR_CODE    = -1
	HTTP_WRONG_FORMAT_ERROR_CODE = -2
	HTTP_SERVICE_ERROR_CODE      = -3
	ERR_UNKNOW_UTXO_CODE         = -4
	ERR_UNKNOW_TX_CODE           = -5
	ERR_ILLEGAL_VIN_CODE         = -6
	ERR_PARAMETERS_CODE          = -7
	ERR_NOT_ENOUGH_BADGE_CODE    = -8
)

type CodeError struct {
	Code int
	Err  error
}

func NewCodeError(Code int, errStr string) error {
	return &CodeError{
		Code: Code,
		Err:  errors.New(errStr),
	}
}

func (this *CodeError) Error() string {
	return this.Err.Error()
}
