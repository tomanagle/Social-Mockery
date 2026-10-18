package profiles

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeJSONAndYAML(t *testing.T) {
	b, _ := json.Marshal(Default())
	for name, fixture := range map[string]string{"json": string(b), "yaml": "version: 1\nclock: 2026-01-01T00:00:00Z\nprocessing_seconds: 5\naccounts:\n  - id: '123'\n    username: example\n    type: BUSINESS\ntokens:\n  - value: token\n    account_id: '123'\n    scopes: [instagram_business_basic]\n"} {
		t.Run(name, func(t *testing.T) {
			s, err := Decode(strings.NewReader(fixture))
			if err != nil {
				t.Fatal(err)
			}
			if len(s.Accounts) != 1 || len(s.Tokens) != 1 || s.ProcessingSeconds != 5 {
				t.Fatalf("unexpected seed: %+v", s)
			}
		})
	}
}
func TestDecodeRejectsInvalidProfiles(t *testing.T) {
	b, _ := json.Marshal(Default())
	valid := string(b)
	cases := map[string]string{"unknown field": strings.TrimSuffix(valid, "}") + `,"typo":true}`, "multiple documents": valid + "\n---\n" + valid, "oversized": strings.Repeat(" ", MaxBytes+1), "empty": "", "invalid JSON": "{", "invalid version": strings.Replace(valid, `"version":1`, `"version":2`, 1), "negative delay": strings.Replace(valid, `"processing_seconds":5`, `"processing_seconds":-1`, 1)}
	for name, v := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode(strings.NewReader(v)); err == nil {
				t.Fatal("accepted invalid profile")
			}
		})
	}
}
func TestValidateRelationships(t *testing.T) {
	for _, name := range []string{"duplicate account", "duplicate token", "missing account", "invalid type", "invalid ID", "invalid clock", "long delay"} {
		t.Run(name, func(t *testing.T) {
			s := Default()
			switch name {
			case "duplicate account":
				s.Accounts = append(s.Accounts, s.Accounts[0])
			case "duplicate token":
				s.Tokens = append(s.Tokens, s.Tokens[0])
			case "missing account":
				s.Tokens[0].AccountID = "999"
			case "invalid type":
				s.Accounts[0].Type = "OTHER"
			case "invalid ID":
				s.Accounts[0].ID = "abc"
			case "invalid clock":
				s.Clock = "yesterday"
			case "long delay":
				s.ProcessingSeconds = 86401
			}
			if Validate(s) == nil {
				t.Fatal("accepted invalid seed")
			}
		})
	}
}
