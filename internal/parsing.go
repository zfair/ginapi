package internal

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func ParamString(c *gin.Context, k string) (string, error) {
	return c.Param(k), nil
}

func ParamBool(c *gin.Context, k string) (bool, error) {
	return strconv.ParseBool(c.Param(k))
}

func ParamInt64(c *gin.Context, k string) (int64, error) {
	return strconv.ParseInt(c.Param(k), 10, 64)
}

func ParamInt32(c *gin.Context, k string) (int32, error) {
	v, err := ParamInt64(c, k)
	return int32(v), err
}

func ParamInt(c *gin.Context, k string) (int, error) {
	v, err := ParamInt64(c, k)
	return int(v), err
}

func ParamUint64(c *gin.Context, k string) (uint64, error) {
	return strconv.ParseUint(c.Param(k), 10, 64)
}

func ParamUint32(c *gin.Context, k string) (uint32, error) {
	v, err := ParamUint64(c, k)
	return uint32(v), err
}

func ParamUint(c *gin.Context, k string) (uint, error) {
	v, err := ParamUint64(c, k)
	return uint(v), err
}

func ParamFloat32(c *gin.Context, k string) (float32, error) {
	v, err := ParamFloat64(c, k)
	return float32(v), err
}

func ParamFloat64(c *gin.Context, k string) (float64, error) {
	return strconv.ParseFloat(c.Param(k), 64)
}
