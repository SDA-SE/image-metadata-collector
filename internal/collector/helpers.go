package collector

import (
	"strconv"
	"strings"
)

func GetOrDefaultBool(m map[string]string, name string, default_ bool) bool {
	var value bool
	value_, success := m[name]
	if success {
		value, _ = strconv.ParseBool(value_)
	} else {
		value = default_
	}
	return value
}

func GetOrDefaultString(m map[string]string, name, default_ string) string {
	value, success := m[name]
	if !success {
		value = default_
	}

	return value
}

func GetOrDefaultInt64(m map[string]string, name string, default_ int64) int64 {
	var value int64
	value_, success := m[name]
	if success {
		value, _ = strconv.ParseInt(value_, 10, 64)
	} else {
		value = default_
	}
	return value
}

func GetOrDefaultStringSlice(m map[string]string, name string, default_ []string) []string {
	var value []string
	value_, success := m[name]
	if success {
		value = strings.Split(value_, ",")
	} else {
		value = default_
	}
	return value
}
