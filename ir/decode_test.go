package ir_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/cantwellnc/eygo/ir"
)

type suiteCase struct {
	Name   string          `json:"name"`
	Source json.RawMessage `json:"source"`
	CID    string          `json:"cid"`
}

// TestIRSuite validates that every node type in the spec's test suite decodes
// without error. Download ir_suite.json from the eyg-lang repo to run this.
func TestIRSuite(t *testing.T) {
	data, err := os.ReadFile("testdata/ir_suite.json")
	if err != nil {
		t.Skip("testdata/ir_suite.json not present:", err)
	}

	var cases []suiteCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal("parse suite:", err)
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			node, err := ir.Decode(tc.Source)
			if err != nil {
				t.Fatalf("Decode(%s): %v", tc.Name, err)
			}
			if node == nil {
				t.Fatal("Decode returned nil node")
			}
		})
	}
}
