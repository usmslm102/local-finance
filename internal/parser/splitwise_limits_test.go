package parser

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
	"testing"
)

func TestSplitwiseBoundedMetadata(t *testing.T) {
	makeCSV := func(names []string, description string) string {
		return "Date,Description,Category,Cost,Currency," + strings.Join(names, ",") + "\n2026-09-01," + description + ",General,100,INR," + strings.TrimSuffix(strings.Repeat("0,", len(names)), ",") + "\n"
	}
	names := []string{"Sanjay Example"}
	for i := 1; i < 100; i++ {
		names = append(names, fmt.Sprintf("Member %03d", i))
	}
	for _, tc := range []struct {
		name, group, person, description string
		members                          []string
		valid                            bool
	}{
		{name: "100 members and field boundaries", group: strings.Repeat("G", 200), person: "Sanjay", description: strings.Repeat("D", 2000), members: names, valid: true},
		{name: "101 members", group: "Demo", person: "Sanjay", description: "Demo", members: append(append([]string(nil), names...), "Extra member")},
		{name: "200 byte member", group: "Demo", person: strings.Repeat("S", 200), description: "Demo", members: []string{strings.Repeat("S", 200)}, valid: true},
		{name: "201 byte member", group: "Demo", person: "Sanjay", description: "Demo", members: []string{"Sanjay", strings.Repeat("A", 201)}},
		{name: "multibyte member limit", group: "Demo", person: "Sanjay", description: "Demo", members: []string{"Sanjay", strings.Repeat("अ", 67)}},
		{name: "case duplicate members", group: "Demo", person: "Sanjay", description: "Demo", members: []string{"Sanjay", "Asha", "ASHA"}},
		{name: "trimmed case duplicate members", group: "Demo", person: "Sanjay", description: "Demo", members: []string{"Sanjay", " Asha ", "asha"}},
		{name: "201 byte group", group: strings.Repeat("G", 201), person: "Sanjay", description: "Demo", members: []string{"Sanjay"}},
		{name: "2001 byte description", group: "Demo", person: "Sanjay", description: strings.Repeat("D", 2001), members: []string{"Sanjay"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := ParseSplitwise(strings.NewReader(makeCSV(tc.members, tc.description)), tc.group, tc.person)
			if tc.valid {
				if err != nil || len(entries) != 1 {
					t.Fatalf("valid boundary rejected: %v", err)
				}
			} else if err == nil {
				t.Fatal("over-limit or ambiguous metadata accepted")
			}
		})
	}
}

func TestSplitwiseExpandedMembershipLimit(t *testing.T) {
	var data bytes.Buffer
	writer := csv.NewWriter(&data)
	header := []string{"Date", "Description", "Category", "Cost", "Currency", "Sanjay"}
	for i := 1; i < 100; i++ {
		header = append(header, fmt.Sprintf("%03d", i)+strings.Repeat("<", 197))
	}
	if err := writer.Write(header); err != nil {
		t.Fatal(err)
	}
	row := []string{"2026-09-01", "Fictional expense", "General", "100", "INR"}
	for i := 0; i < 100; i++ {
		row = append(row, "0")
	}
	for i := 0; i < 160; i++ {
		if err := writer.Write(row); err != nil {
			t.Fatal(err)
		}
	}
	writer.Flush()
	if data.Len() >= int(MaxSplitwiseFileSize) {
		t.Fatal("test must exercise small input expansion")
	}
	_, err := ParseSplitwise(&data, "Demo", "Sanjay")
	if err == nil || !strings.Contains(err.Error(), "expanded membership limit") {
		t.Fatalf("JSON-escaped metadata expansion accepted: %v", err)
	}
}
