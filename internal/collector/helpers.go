package collector

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// GetOrDefaultBool returns the value of the given name from the map m or the default value if it doesn't exist.
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

// GetOrDefaultString returns the value of the given name from the map m or the default value if it doesn't exist.
func GetOrDefaultString(m map[string]string, name, default_ string) string {
	value, success := m[name]
	if !success {
		value = default_
	}

	if len(value) == 0 {
		value = default_
	}

	return value
}

// GetOrDefaultInt64 the value of the given name from the map m or the default value if it doesn't exist.
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

// GetOrDefaultStringSlice the value of the given name from the map m or the default value if it doesn't exist.
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

func GetOrDefaultOwners(tags map[string]string, key string, defaultValue []Owner) []Owner {
	value, ok := tags[key]
	if !ok || value == "" {
		return defaultValue
	}

	var owners []Owner
	if err := json.Unmarshal([]byte(value), &owners); err != nil {
		log.Warn().Err(err).Msgf("Could not parse owners from tag %s", key)
		return defaultValue
	}

	return owners
}

func GetOrDefaultNotifications(tags map[string]string, key string, defaultValue Notifications) Notifications {
	value, ok := tags[key]
	if !ok || value == "" {
		return defaultValue
	}

	var notifications Notifications
	if err := json.Unmarshal([]byte(value), &notifications); err != nil {
		log.Warn().Err(err).Msgf("Could not parse notifications from tag %s", key)
		return defaultValue
	}

	return notifications
}

type JsonMarshal func(any) ([]byte, error)

func JsonIndentMarshal(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "\t")
}
