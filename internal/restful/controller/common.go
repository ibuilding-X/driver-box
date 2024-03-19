package controller

import (
	"encoding/json"
	"github.com/ibuilding-x/driver-box/internal/restful/response"
	"net/http"
)

// commonController 通用控制器
type commonController struct {
}

func (c *commonController) Success(w http.ResponseWriter, data any) {
	c.JSON(w, http.StatusOK, response.Common{
		Message: "success",
		Data:    data,
	})
}

func (c *commonController) Error(w http.ResponseWriter, code int, err error, data any) {
	c.JSON(w, code, response.Common{
		Message: err.Error(),
		Data:    data,
	})
}

func (c *commonController) JSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	b, _ := json.Marshal(v)
	_, _ = w.Write(b)
}
