package collector

import (
	"encoding/json"
	"sort"
	"testing"
)

var testMap = map[string]string{
	"str":          "some-string",
	"int64":        "123",
	"bool":         "true",
	"string-slice": "a,b,c",
	"float":        "1.23",
	"empty":        "",
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
		TestCaseHelper{
			name:           "EmptyString",
			inputMap:       testMap,
			targetKeyName:  "empty",
			targetDefault:  "default-value",
			expectedResult: "default-value",
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

func TestGetOrDefaultOwners(t *testing.T) {
	owners := []Owner{
		{Role: "admin", Uuid: "1234", Name: "Alice"},
		{Role: "viewer", Uuid: "5678", Name: "Bob"},
	}
	testMap := map[string]string{"owners": "[{\"role\":\"admin\",\"uuid\":\"1234\",\"name\":\"Alice\"},{\"role\":\"viewer\",\"uuid\":\"5678\",\"name\":\"Bob\"}]"}

	testCases := []TestCaseHelper{
		{
			name:           "ValidJSON",
			inputMap:       testMap,
			targetKeyName:  "owners",
			targetDefault:  []Owner{},
			expectedResult: owners,
		},
		{
			name:           "KeyMissing_ReturnsDefault",
			inputMap:       testMap,
			targetKeyName:  "invalidkey",
			targetDefault:  []Owner{},
			expectedResult: []Owner{},
		},
		{
			name:           "EmptyValue_ReturnsDefault",
			inputMap:       map[string]string{"owners": ""},
			targetKeyName:  "owners",
			targetDefault:  []Owner{{Role: "default", Uuid: "0000", Name: "Default"}},
			expectedResult: []Owner{{Role: "default", Uuid: "0000", Name: "Default"}},
		},
		{
			name:           "InvalidJSON_ReturnsDefault",
			inputMap:       map[string]string{"owners": "not-valid-json"},
			targetKeyName:  "owners",
			targetDefault:  []Owner{{Role: "fallback", Uuid: "9999", Name: "Fallback"}},
			expectedResult: []Owner{{Role: "fallback", Uuid: "9999", Name: "Fallback"}},
		},
		{
			name:           "EmptyArray",
			inputMap:       map[string]string{"owners": "[]"},
			targetKeyName:  "owners",
			targetDefault:  []Owner{{Role: "fallback", Uuid: "9999", Name: "Fallback"}},
			expectedResult: []Owner{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetOrDefaultOwners(tc.inputMap, tc.targetKeyName, tc.targetDefault.([]Owner))
			expectedResult := tc.expectedResult.([]Owner)
			if len(expectedResult) != len(result) {
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

// --- GetOrDefaultNotifications ---
func TestGetOrDefaultNotifications_ValidJSON(t *testing.T) {
	notifications := Notifications{
		Slack:   []string{"#channel-a", "#channel-b"},
		Emails:  []string{"alice@example.com", "bob@example.com"},
		MSTeams: []string{"team-a", "team-b"},
	}
	data, _ := json.Marshal(notifications)
	m := map[string]string{"notifications": string(data)}

	got := GetOrDefaultNotifications(m, "notifications", Notifications{})
	if len(got.Slack) != 2 || got.Slack[0] != "#channel-a" || got.Slack[1] != "#channel-b" {
		t.Errorf("unexpected Slack: %v", got.Slack)
	}
	if len(got.Emails) != 2 || got.Emails[0] != "alice@example.com" {
		t.Errorf("unexpected Emails: %v", got.Emails)
	}
	if len(got.MSTeams) != 2 || got.MSTeams[0] != "team-a" {
		t.Errorf("unexpected MSTeams: %v", got.MSTeams)
	}
}

func TestGetOrDefaultNotifications_KeyMissing_ReturnsDefault(t *testing.T) {
	m := map[string]string{}
	def := Notifications{Slack: []string{"#default"}}
	got := GetOrDefaultNotifications(m, "notifications", def)
	if len(got.Slack) != 1 || got.Slack[0] != "#default" {
		t.Errorf("expected default notifications, got %v", got)
	}
}

func TestGetOrDefaultNotifications_EmptyValue_ReturnsDefault(t *testing.T) {
	m := map[string]string{"notifications": ""}
	def := Notifications{Emails: []string{"fallback@example.com"}}
	got := GetOrDefaultNotifications(m, "notifications", def)
	if len(got.Emails) != 1 || got.Emails[0] != "fallback@example.com" {
		t.Errorf("expected default notifications for empty value, got %v", got)
	}
}

func TestGetOrDefaultNotifications_InvalidJSON_ReturnsDefault(t *testing.T) {
	m := map[string]string{"notifications": "not-valid-json"}
	def := Notifications{MSTeams: []string{"fallback-team"}}
	got := GetOrDefaultNotifications(m, "notifications", def)
	if len(got.MSTeams) != 1 || got.MSTeams[0] != "fallback-team" {
		t.Errorf("expected default notifications on invalid JSON, got %v", got)
	}
}

func TestGetOrDefaultNotifications_PartialJSON(t *testing.T) {
	m := map[string]string{"notifications": `{"slack":["#only-slack"]}`}
	got := GetOrDefaultNotifications(m, "notifications", Notifications{})
	if len(got.Slack) != 1 || got.Slack[0] != "#only-slack" {
		t.Errorf("unexpected Slack: %v", got.Slack)
	}
	if len(got.Emails) != 0 {
		t.Errorf("expected empty Emails, got %v", got.Emails)
	}
	if len(got.MSTeams) != 0 {
		t.Errorf("expected empty MSTeams, got %v", got.MSTeams)
	}
}

func TestGetOrDefaultNotifications_EmptyArrays(t *testing.T) {
	m := map[string]string{"notifications": `{"slack":[],"emails":[],"ms_teams":[]}`}
	got := GetOrDefaultNotifications(m, "notifications", Notifications{Slack: []string{"#default"}})
	if len(got.Slack) != 0 {
		t.Errorf("expected empty Slack, got %v", got.Slack)
	}
}

// --- JsonIndentMarshal ---
func TestJsonIndentMarshal_ValidInput(t *testing.T) {
	input := []Owner{{Role: "admin", Uuid: "1234", Name: "Alice"}}
	data, err := JsonIndentMarshal(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
	// Verify it's valid JSON by unmarshalling back
	var result []Owner
	if err := json.Unmarshal(data, &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
}

func TestJsonIndentMarshal_EmptySlice(t *testing.T) {
	data, err := JsonIndentMarshal([]Owner{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "[]" {
		t.Errorf("expected '[]', got %s", string(data))
	}
}
