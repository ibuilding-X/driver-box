package response

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// JSON 返回 json 数据
func JSON(w http.ResponseWriter, code int, obj any) {
	s, err := json.Marshal(obj)
	if err != nil {
		msg := fmt.Sprintf("json encode error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(msg))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(s)
}

// String 返回 string 数据
func String(w http.ResponseWriter, code int, format string, values ...any) {
	s := fmt.Sprintf(format, values...)
	w.WriteHeader(code)
	_, _ = w.Write([]byte(s))
}
