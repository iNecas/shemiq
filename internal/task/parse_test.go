package task

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/iNecas/shemiq/internal/utils"
)

func TestParseMarkdownDocumentSections(t *testing.T) {
	data := []byte(utils.Dedent(`
		:::shemiq
		status: typo
		type: unknown
		uuid: invalid
		unknown-key_2: keep: colons
		status: done
		empty:
		:::
		# **Main** ##
		prose

		:::shemiq
		type: top-level
		:::
		### Skipped
		:::shemiq
		:::
		#### Child
		:::shemiq
		status:
		:::
		## Sibling
		:::shemiq
		type: task
		:::
		# Last
	`))
	doc, err := parseMarkdownDocument("relative/../example.md", data)
	if err != nil {
		t.Fatal(err)
	}
	root := doc.root
	if doc.path != "relative/../example.md" || string(doc.data) != string(data) ||
		root.level != 0 || root.span != (sourceSpan{0, len(data), 1}) {
		t.Fatalf("unexpected document/root: %+v, %+v", doc, root)
	}
	if len(root.directives) != 1 || len(root.children) != 2 {
		t.Fatalf("root ownership: %+v", root)
	}
	fields := root.directives[0].fields
	wantFields := [][2]string{{"status", "typo"}, {"type", "unknown"}, {"uuid", "invalid"},
		{"unknown-key_2", "keep: colons"}, {"status", "done"}, {"empty", ""}}
	var gotFields [][2]string
	for _, field := range fields {
		gotFields = append(gotFields, [2]string{field.key, field.value})
	}
	if !reflect.DeepEqual(gotFields, wantFields) {
		t.Fatalf("field occurrences: got %v, want %v", gotFields, wantFields)
	}
	main, last := root.children[0], root.children[1]
	if main.level != 1 || main.title != "**Main**" || len(main.children) != 2 ||
		len(main.directives) != 1 || last.title != "Last" {
		t.Fatalf("unexpected first-level sections: %+v, %+v", main, last)
	}
	skipped, sibling := main.children[0], main.children[1]
	if skipped.level != 3 || skipped.title != "Skipped" || len(skipped.children) != 1 ||
		len(skipped.directives) != 1 || len(skipped.directives[0].fields) != 0 {
		t.Fatalf("skipped-level/empty-directive structure: %+v", skipped)
	}
	child := skipped.children[0]
	if child.level != 4 || child.title != "Child" || len(child.directives) != 1 ||
		child.directives[0].fields[0].key != "status" || sibling.level != 2 || len(sibling.directives) != 1 {
		t.Fatalf("nested directive ownership: %+v, %+v", child, sibling)
	}
	for _, check := range []struct {
		section *parsedSection
		start   string
		end     int
	}{
		{main, "# **Main**", last.heading.start},
		{skipped, "### Skipped", sibling.heading.start},
		{child, "#### Child", sibling.heading.start},
		{sibling, "## Sibling", last.heading.start},
		{last, "# Last", len(data)},
	} {
		section := check.section
		if section.span.start != strings.Index(string(data), check.start) ||
			section.span.end != check.end || section.heading.start != section.span.start ||
			section.heading.line != section.span.line {
			t.Errorf("section boundaries: %+v", section)
		}
	}
	gap := string(data[main.heading.end:main.directives[0].opening.start])
	if gap != "prose\n\n" {
		t.Fatalf("heading/directive gap not retained: %q", gap)
	}
}

func TestParseMarkdownDocumentMarkdownSubset(t *testing.T) {
	for _, tc := range []struct {
		name       string
		data       string
		titles     []string
		levels     []int
		directives int
	}{
		{"empty", "", nil, nil, 0},
		{"headingless metadata", utils.Dedent(`
			:::shemiq
			status: new
			:::`), nil, nil, 1},
		{"ATX headings", utils.Dedent(`
			# Title ###` + " \t" + `
			##` + "\t" + `*Literal* ##
			######
			# ###
			# Has###
			#not-heading
			####### too many
			 # indented
			Setext
			======
			`), []string{"Title", "*Literal*", "", "", "Has###"}, []int{1, 2, 6, 1, 1}, 0},
		{"backtick fence rules", "   " + utils.Dedent(`
			`+"````go"+`
			# Hidden
			:::shemiq invalid
			`+"```"+`
			~~~~
			`+"```` not a closer"+`
			# Also hidden
			`+"  ````` \t"+`
			# Visible
			:::shemiq
			:::
			`), []string{"Visible"}, []int{1}, 1},
		{"tilde fence rules", "  " + utils.Dedent(`
			~~~~ info `+"`allowed`"+`
			# Hidden
			:::shemiq
			not a field
			:::shemiq
			~~~
			   ~~~~~`+"\t"+`
			## Visible
			`), []string{"Visible"}, []int{2}, 0},
		{"unclosed fence", utils.Dedent(`
			# Visible
			` + "```" + `
			## Hidden
			:::shemiq malformed
			`), []string{"Visible"}, []int{1}, 0},
		{"backticks in info are not an opener", utils.Dedent(`
			` + "```bad`info" + `
			# Visible
			:::shemiq
			:::
			`), []string{"Visible"}, []int{1}, 1},
		{"indented examples and stray closer", "    " + utils.Dedent(`
			`+"```"+`
			 # Not a heading
			 :::shemiq malformed
			    :::shemiq
			    invalid field
			    :::
			:::
			# Visible
			`), []string{"Visible"}, []int{1}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := parseMarkdownDocument("example.md", []byte(tc.data))
			if err != nil {
				t.Fatal(err)
			}
			var titles []string
			var levels []int
			directives := 0
			var visit func(*parsedSection)
			visit = func(section *parsedSection) {
				if section.level != 0 {
					titles = append(titles, section.title)
					levels = append(levels, section.level)
				}
				directives += len(section.directives)
				for _, child := range section.children {
					visit(child)
				}
			}
			visit(doc.root)
			if !reflect.DeepEqual(titles, tc.titles) || !reflect.DeepEqual(levels, tc.levels) || directives != tc.directives {
				t.Fatalf("got titles %q, levels %v, directives %d; want %q, %v, %d", titles, levels, directives, tc.titles, tc.levels, tc.directives)
			}
		})
	}
}

func TestParseMarkdownDocumentSyntaxErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		data    string
		line    int
		message string
	}{
		{"malformed opener", utils.Dedent(`
			text
			:::shemiq extra
			`), 2, "malformed shemiq directive opener"},
		{"opener trailing space", utils.Dedent(`
			:::shemiq` + " " + `
			`), 1, "malformed shemiq directive opener"},
		{"nested directive", utils.Dedent(`
			:::shemiq
			:::shemiq
			`), 2, "nested shemiq directive"},
		{"malformed nested opener", utils.Dedent(`
			:::shemiq
			:::shemiq extra
			`), 2, "malformed shemiq directive opener"},
		{"blank field", utils.Dedent(`
			:::shemiq

			:::
			`), 2, "malformed metadata field"},
		{"missing colon", utils.Dedent(`
			:::shemiq
			status
			:::
			`), 2, "malformed metadata field"},
		{"indented field", utils.Dedent(`
			:::shemiq
			 status: new
			:::
			`), 2, "malformed metadata field"},
		{"numeric key", utils.Dedent(`
			:::shemiq
			1key: new
			:::
			`), 2, "malformed metadata field"},
		{"non-ASCII key", utils.Dedent(`
			:::shemiq
			clé: value
			:::
			`), 2, "malformed metadata field"},
		{"indented closer", utils.Dedent(`
			:::shemiq
			 :::
			`), 2, "malformed metadata field"},
		{"fence inside directive", utils.Dedent(`
			:::shemiq
			` + "```" + `
			not a field
			`), 2, "malformed metadata field"},
		{"unterminated directive", utils.Dedent(`
			text
			:::shemiq
			status: new`), 2, "unterminated shemiq directive"},
		{"no partial document", utils.Dedent(`
			# Valid
			:::shemiq
			:::
			:::shemiq broken
			`), 4, "malformed shemiq directive opener"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := parseMarkdownDocument("relative/../example.md", []byte(tc.data))
			want := fmt.Sprintf("relative/../example.md:%d: %s", tc.line, tc.message)
			if doc != nil || err == nil || err.Error() != want {
				t.Fatalf("got document %v, error %v; want nil document, %q", doc, err, want)
			}
		})
	}
}

func TestParseMarkdownDocumentSourceEdits(t *testing.T) {
	for _, tc := range []struct {
		name          string
		headingEnding string
		openingEnding string
		fieldEnding   string
		lastEnding    string
		closingEnding string
		rawValue      string
		editedValue   string
	}{
		{"LF", "\n", "\n", "\n", "\n", "\n", " \trefined \t", " \tdone \t"},
		{"CRLF", "\r\n", "\r\n", "\r\n", "\r\n", "\r\n", " \trefined \t", " \tdone \t"},
		{"mixed prefers CRLF", "\n", "\r\n", "\n", "\r\n", "\n", " \trefined \t", " \tdone \t"},
		{"mixed prefers LF", "\r\n", "\n", "\r\n", "\n", "\r\n", " \trefined \t", " \tdone \t"},
		{"closer at EOF", "\r\n", "\r\n", "\r\n", "\r\n", "", " \trefined \t", " \tdone \t"},
		{"empty value", "\n", "\n", "\n", "\n", "", "", "done"},
		{"whitespace-only value", "\n", "\n", "\n", "\n", "", " \t", "done \t"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := "# Títle" + tc.headingEnding + "\tuntouched  " + tc.headingEnding + ":::shemiq" + tc.openingEnding
			afterField := "unknown-key_2: keep: extra" + tc.lastEnding
			tail := ":::" + tc.closingEnding
			if tc.closingEnding != "" {
				tail += "tail  \t"
			}
			original := before + "status:" + tc.rawValue + tc.fieldEnding + afterField + tail
			data := []byte(original)
			doc, err := parseMarkdownDocument("example.md", data)
			if err != nil {
				t.Fatal(err)
			}
			section := doc.root.children[0]
			directive := section.directives[0]
			field := directive.fields[0]
			opening := sourceSpan{strings.Index(original, ":::shemiq"), len(before), 3}
			closing := sourceSpan{len(original) - len(tail), len(original) - len(tail) + len(":::") + len(tc.closingEnding), 6}
			if section.heading != (sourceSpan{0, len("# Títle") + len(tc.headingEnding), 1}) || directive.opening != opening || directive.closing != closing || directive.span != (sourceSpan{opening.start, closing.end, 3}) || directive.insertAt != closing.start || directive.lineEnding != tc.lastEnding {
				t.Fatalf("incorrect heading/directive positions: %+v, %+v", section.heading, directive)
			}
			valueStart := len(before) + len("status:")
			if field.value != "" {
				valueStart += strings.Index(tc.rawValue, field.value)
			}
			if field.span != (sourceSpan{len(before), len(before) + len("status:") + len(tc.rawValue) + len(tc.fieldEnding), 4}) || field.keySpan != (sourceSpan{len(before), len(before) + len("status"), 4}) || field.valueSpan != (sourceSpan{valueStart, valueStart + len(field.value), 4}) {
				t.Fatalf("incorrect field positions: %+v", field)
			}
			for _, field := range directive.fields {
				if string(data[field.keySpan.start:field.keySpan.end]) != field.key || string(data[field.valueSpan.start:field.valueSpan.end]) != field.value {
					t.Fatalf("spans do not address original key/value bytes: %+v", field)
				}
			}
			// Apply both edits against original offsets, without production repair code.
			edited := string(data[:field.valueSpan.start]) + "done" + string(data[field.valueSpan.end:directive.insertAt]) + "uuid: generated" + directive.lineEnding + string(data[directive.insertAt:])
			want := before + "status:" + tc.editedValue + tc.fieldEnding + afterField + "uuid: generated" + tc.lastEnding + tail
			if edited != want || string(doc.data) != original {
				t.Fatalf("edit changed unrelated bytes or original source:\n got %q\nwant %q", edited, want)
			}
			reparsed, err := parseMarkdownDocument(doc.path, []byte(edited))
			if err != nil {
				t.Fatal(err)
			}
			fields := reparsed.root.children[0].directives[0].fields
			if len(fields) != 3 || fields[0].value != "done" || fields[2].key != "uuid" || fields[2].value != "generated" {
				t.Fatalf("edited metadata not retained: %+v", fields)
			}
		})
	}
}
