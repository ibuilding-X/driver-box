package restful

import "errors"

var (
	// UndefinedErr 未定义错误
	UndefinedErr = errors.New("undefined error")
)

// errorCodes maps error to its corresponding internal status code.
var errorCodes = map[error]int{
	UndefinedErr: 500, // 未定义错误
}
