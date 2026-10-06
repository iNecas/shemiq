package task

import (
	"strings"
	"testing"
)

func TestSectionTaskStatusConversion(t *testing.T) {
	for _, tc := range []struct {
		name, fields, status, message string
		line                          int
	}{
		{"omitted", "", "new", "missing uuid", 2},
		{"invalid", "status: refiend\n", "", "invalid status", 4},
		{"empty", "status: \n", "", "empty metadata field: status", 4},
		{"repeated", "status: new\nstatus: done\n", "", "repeated metadata field: status", 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.name + ".md"
			data := []byte("# Title\n:::shemiq\ntype: top-level\n" + tc.fields + ":::\n")
			doc, err := parseMarkdownDocument(path, data)
			if err != nil {
				t.Fatal(err)
			}
			got := sectionTask(doc, doc.root.children[0], false)
			if got.Status != tc.status || got.Title != "Title" {
				t.Fatalf("conversion: task=%+v", got)
			}
			found := false
			for _, issue := range got.Issues {
				if issue.Path == path && issue.Line == tc.line &&
					strings.Contains(issue.Message, tc.message) {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing located issue %q: %+v", tc.message, got.Issues)
			}
		})
	}
}

func TestDirectiveFieldsMetadataOnly(t *testing.T) {
	for _, status := range []string{"new", "invalid"} {
		t.Run(status, func(t *testing.T) {
			path := "metadata.md"
			doc, err := parseMarkdownDocument(path,
				[]byte(":::shemiq\nstatus: "+status+"\n:::\n"))
			if err != nil {
				t.Fatal(err)
			}
			fields, issues := directiveFields(path, doc.root.directives[0])
			if status == "invalid" {
				if fields["status"].value != "" || len(issues) != 2 ||
					issues[0].Message != "invalid status: invalid" ||
					issues[1].Message != "missing uuid" {
					t.Fatalf("metadata-only conversion: fields=%+v issues=%+v", fields, issues)
				}
			} else if fields["status"].value != "new" || len(issues) != 1 ||
				issues[0].Message != "missing uuid" {
				t.Fatalf("query structure leaked into general metadata findings: %+v", issues)
			}
		})
	}
}
