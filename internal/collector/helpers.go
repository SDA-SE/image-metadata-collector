package collector

import (
	"strconv"
	"strings"
)

func GetOrDefaultBool(m map[string]string, name string, default_ bool) bool {
	var value bool
	var err error

	value_, success := m[name]

	if success {
		value, err = strconv.ParseBool(value_)
	}

	if !success || err != nil {
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
	var err error

	value_, success := m[name]
	if success {
		value, err = strconv.ParseInt(value_, 10, 64)
	}

	if !success || err != nil {
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
