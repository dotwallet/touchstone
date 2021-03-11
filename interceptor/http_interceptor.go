package interceptor

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/dotwallet/touchstone/util"
	"github.com/golang/glog"
)

const (
	HTTP_OK_RESPONSE_CODE        = 0
	HTTP_READ_BODY_ERROR_CODE    = -1
	HTTP_WRONG_FORMAT_ERROR_CODE = -2
	HTTP_SERVICE_ERROR_CODE      = -3
)

type HttpReqBody interface {
	NewHttpReqBody() HttpReqBody
}

type HttpJsonResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

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

func NewHttpJsonResponse(code int, msg string, data interface{}) []byte {
	httpJsonResponse := &HttpJsonResponse{
		Code: code,
		Msg:  msg,
		Data: data,
	}
	b, err := json.Marshal(httpJsonResponse)
	if err != nil {
		return []byte(err.Error())
	}
	return b
}

func NewOkHttpJsonResponse(data interface{}) []byte {
	return NewHttpJsonResponse(HTTP_OK_RESPONSE_CODE, "", data)
}
func NewErrHttpJsonResponse(code int, msg string) []byte {
	return NewHttpJsonResponse(code, msg, nil)
}

func DoCheckNilRecursion(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return errors.New("required fields are missing")
		}
		err := DoCheckNilRecursion(v.Elem())
		if err != nil {
			return errors.New("required fields are missing")
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			err := DoCheckNilRecursion(v.Field(i))
			if err != nil {
				return errors.New("required fields are missing")
			}
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			err := DoCheckNilRecursion(v.Index(i))
			if err != nil {
				return errors.New("required fields are missing")
			}
		}
	}
	return nil
}

func CheckNil(inter interface{}) error {
	v := reflect.ValueOf(inter)
	return DoCheckNilRecursion(v)
}

func Aspect(
	handleFunc func(rsp http.ResponseWriter, req *http.Request, httpReqStruct HttpReqBody, reqid string) (interface{}, error),
	httpReqBody HttpReqBody,
) func(rsp http.ResponseWriter, req *http.Request) {
	return func(rsp http.ResponseWriter, req *http.Request) {
		reqid := util.RandStringBytes(8)
		bodyBytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			rsp.Write(NewErrHttpJsonResponse(HTTP_READ_BODY_ERROR_CODE, err.Error()))
			return
		}
		glog.Infof("Aspect %s %s %s", req.URL.String(), string(bodyBytes), reqid)
		reqBody := httpReqBody.NewHttpReqBody()
		if reqBody != nil {
			err = json.Unmarshal(bodyBytes, reqBody)
			if err != nil {
				rsp.Write(NewErrHttpJsonResponse(HTTP_WRONG_FORMAT_ERROR_CODE, err.Error()))
				return
			}
			err = CheckNil(reqBody)
			if err != nil {
				rsp.Write(NewErrHttpJsonResponse(HTTP_WRONG_FORMAT_ERROR_CODE, err.Error()))
				return
			}
		}
		result, err := handleFunc(rsp, req, reqBody, reqid)
		if err != nil {
			codeErr, ok := err.(*CodeError)
			if !ok {
				rsp.Write(NewErrHttpJsonResponse(HTTP_SERVICE_ERROR_CODE, err.Error()))
				return
			}
			rsp.Write(NewErrHttpJsonResponse(codeErr.Code, codeErr.Error()))
			return
		}
		rsp.Write(NewOkHttpJsonResponse(result))
	}
}
