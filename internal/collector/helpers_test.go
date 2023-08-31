package collector

import (
	"sort"
	"testing"
)

var testMap = map[string]string{
	"str":          "some-string",
	"int64":        "123",
	"bool":         "true",
	"string-slice": "a,b,c",
	"float":        "1.23",
}

type TestCaseHelper struct {
	name           string
	inputMap       map[string]string
	targetKeyName  string
	targetDefault  interface{}
	expectedResult interface{}
}

func TestGetOrDefaultBool(t *testing.T) {
	testCases := []TestCaseHelper{
		TestCaseHelper{
			name:           "MissingKeyExpectDefault",
			inputMap:       testMap,
			targetKeyName:  "does-not-exist",
			targetDefault:  true,
			expectedResult: true,
		},
		TestCaseHelper{
			name:           "ExistingKeyWrongStrTypeExpectDefault",
			inputMap:       testMap,
			targetKeyName:  "str",
			targetDefault:  true,
			expectedResult: true,
		},
		TestCaseHelper{
			name:           "ExistingKeyWrongIntTypeExpectDefault",
			inputMap:       testMap,
			targetKeyName:  "int64",
			targetDefault:  true,
			expectedResult: true,
		},
		TestCaseHelper{
			name:           "ExistingKeyWrongStringSliceTypeExpectDefault",
			inputMap:       testMap,
			targetKeyName:  "string-slice",
			targetDefault:  true,
			expectedResult: true,
		},
		TestCaseHelper{
			name:           "ExistingKeyWrongFloatTypeExpectDefault",
			inputMap:       testMap,
			targetKeyName:  "float",
			targetDefault:  true,
			expectedResult: true,
		},
		TestCaseHelper{
			name:           "ExistingKey",
			inputMap:       testMap,
			targetKeyName:  "bool",
			targetDefault:  false,
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetOrDefaultBool(tc.inputMap, tc.targetKeyName, tc.targetDefault.(bool))
			if result != tc.expectedResult.(bool) {
				t.Fatalf("Expected %v, got %v", tc.expectedResult, result)
			}
		})
	}
}

func TestGetOrDefaultString(t *testing.T) {
	testCases := []TestCaseHelper{
		TestCaseHelper{
			name:           "MissingKeyExpectDefault",
			inputMap:       testMap,
			targetKeyName:  "does-not-exist",
			targetDefault:  "default-value",
			expectedResult: "default-value",
		},
		// Everything is a string and there is no good way of saying "123" is not a string.
		TestCaseHelper{
			name:           "ExistingKeyWrongTypeExpectKeyValue",
			inputMap:       testMap,
			targetKeyName:  "int64",
			targetDefault:  "default-value",
			expectedResult: "123",
		},
		TestCaseHelper{
			name:           "ExistingKey",
			inputMap:       testMap,
			targetKeyName:  "str",
			targetDefault:  "default-value",
			expectedResult: "some-string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetOrDefaultString(tc.inputMap, tc.targetKeyName, tc.targetDefault.(string))
			if result != tc.expectedResult.(string) {
				t.Fatalf("Expected %v, got %v", tc.expectedResult, result)
			}
		})
	}
}

func TestGetOrDefaultInt(t *testing.T) {
	testCases := []TestCaseHelper{
		TestCaseHelper{
			name:           "MissingKeyExpectDefault",
			inputMap:       testMap,
			targetKeyName:  "does-not-exist",
			targetDefault:  int64(999),
			expectedResult: int64(999),
		},
		TestCaseHelper{
			name:           "ExistingKeyWrongStrTypeExpectDefault",
			inputMap:       testMap,
			targetKeyName:  "str",
			targetDefault:  int64(999),
			expectedResult: int64(999),
		},
		TestCaseHelper{
			name:           "ExistingKeyWrongBoolTypeExpectDefault",
			inputMap:       testMap,
			targetKeyName:  "bool",
			targetDefault:  int64(999),
			expectedResult: int64(999),
		},
		TestCaseHelper{
			name:           "ExistingKeyWrongStringSliceTypeExpectDefault",
			inputMap:       testMap,
			targetKeyName:  "string-slice",
			targetDefault:  int64(999),
			expectedResult: int64(999),
		},
		TestCaseHelper{
			name:           "ExistingKeyWrongFloatTypeExpectDefault",
			inputMap:       testMap,
			targetKeyName:  "float",
			targetDefault:  int64(999),
			expectedResult: int64(999),
		},
		TestCaseHelper{
			name:           "ExistingKey",
			inputMap:       testMap,
			targetKeyName:  "int64",
			targetDefault:  int64(999),
			expectedResult: int64(123),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetOrDefaultInt64(tc.inputMap, tc.targetKeyName, tc.targetDefault.(int64))
			if result != tc.expectedResult.(int64) {
				t.Fatalf("Expected %v, got %v", tc.expectedResult, result)
			}
		})
	}
}

func TestGetOrDefaultStringSlice(t *testing.T) {
	testCases := []TestCaseHelper{
		TestCaseHelper{
			name:           "MissingKeyExpectDefault",
			inputMap:       testMap,
			targetKeyName:  "does-not-exist",
			targetDefault:  []string{"foo", "bar"},
			expectedResult: []string{"foo", "bar"},
		},
		// Everything is a string and there is no good way of saying "123" is not a string.
		TestCaseHelper{
			name:           "ExistingKeyWrongStrTypeExpectKeyValue",
			inputMap:       testMap,
			targetKeyName:  "str",
			targetDefault:  []string{"foo", "bar"},
			expectedResult: []string{"some-string"},
		},
		// Everything is a string and there is no good way of saying "123" is not a string.
		TestCaseHelper{
			name:           "ExistingKeyWrongIntTypeExpectKeyValue",
			inputMap:       testMap,
			targetKeyName:  "int64",
			targetDefault:  []string{"foo", "bar"},
			expectedResult: []string{"123"},
		},
		// Everything is a string and there is no good way of saying "123" is not a string.
		TestCaseHelper{
			name:           "ExistingKeyWrongBoolTypeExpectKeyValue",
			inputMap:       testMap,
			targetKeyName:  "bool",
			targetDefault:  []string{"foo", "bar"},
			expectedResult: []string{"true"},
		},
		// Everything is a string and there is no good way of saying "123" is not a string.
		TestCaseHelper{
			name:           "ExistingKeyWrongFloatTypeExpectKeyValue",
			inputMap:       testMap,
			targetDefault:  []string{"foo", "bar"},
			targetKeyName:  "float",
			expectedResult: []string{"1.23"},
		},
		TestCaseHelper{
			name:           "ExistingKey",
			inputMap:       testMap,
			targetKeyName:  "string-slice",
			targetDefault:  []string{"foo", "bar"},
			expectedResult: []string{"a", "b", "c"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetOrDefaultStringSlice(tc.inputMap, tc.targetKeyName, tc.targetDefault.([]string))
			sort.Slice(result, func(i, j int) bool {
				return result[i] < result[j]
			})

			expectedResult := tc.expectedResult.([]string)
			sort.Slice(expectedResult, func(i, j int) bool {
				return expectedResult[i] < expectedResult[j]
			})

			if len(result) != len(expectedResult) {
				t.Fatalf("Expected %v, got %v, length missmatching", tc.expectedResult, result)
			}

			for i, v := range result {
				if expectedResult[i] != v {
					t.Fatalf("Expected %v, got %v, value missmatching (%v != %v)", expectedResult, result, expectedResult[i], v)
				}
			}

		})
	}
}
