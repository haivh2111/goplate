package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/haivh2111/goplate/internal/fields"
	"github.com/haivh2111/goplate/internal/naming"
	"github.com/haivh2111/goplate/internal/templates"
)

// EventOptions configures a single `goplate new-event` run.
type EventOptions struct {
	EventName   string // PascalCase, e.g. "OrderPlaced"
	PayloadSpec string // raw --payload DSL string; may be empty
	Subscriber  string // optional feature name that handles the event
	ModulePath  string // Go module path of the target project
	ProjectRoot string // absolute path; used to detect existing files
}

type eventData struct {
	EventName    string         // "OrderPlaced"
	EventTopic   string         // "order.placed"
	Subscriber   string         // "notification" or ""
	ModulePath   string         // "github.com/acme/svc"
	Fields       []fields.Field // payload fields
	HasTimeField bool
}

var eventNameRE = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

// GenerateEvent validates opts, parses the payload DSL, and returns the file
// list for an event run plus a human-readable notice (empty for fresh runs).
func GenerateEvent(opts EventOptions) ([]File, string, error) {
	if !eventNameRE.MatchString(opts.EventName) {
		return nil, "", fmt.Errorf("event name %q is invalid (must match ^[A-Z][a-zA-Z0-9]*$)", opts.EventName)
	}
	if opts.ModulePath == "" {
		return nil, "", fmt.Errorf("module path is required (run from a Go project root)")
	}
	var parsed []fields.Field
	if strings.TrimSpace(opts.PayloadSpec) != "" {
		ps, err := fields.Parse(opts.PayloadSpec)
		if err != nil {
			return nil, "", fmt.Errorf("--payload: %w", err)
		}
		parsed = ps
	}
	if opts.Subscriber != "" {
		if err := validateLowerName("subscriber", opts.Subscriber); err != nil {
			return nil, "", err
		}
	}

	data := eventData{
		EventName:    opts.EventName,
		EventTopic:   naming.ToDotted(opts.EventName),
		Subscriber:   opts.Subscriber,
		ModulePath:   opts.ModulePath,
		Fields:       parsed,
		HasTimeField: fields.NeedsTimeImport(parsed),
	}

	var (
		out      []File
		notices  []string
		eventRel = path.Join("internal", "events", "event_types.go")
	)

	// internal/events/event_types.go — create or append.
	eventAbs := filepath.Join(opts.ProjectRoot, filepath.FromSlash(eventRel))
	if existing, ok := readExisting(eventAbs); ok {
		merged, alreadyPresent, err := appendEventTypes(existing, data)
		if err != nil {
			return nil, "", fmt.Errorf("append event_types.go: %w", err)
		}
		if alreadyPresent {
			notices = append(notices, fmt.Sprintf("  ↻  %s: %s already declared — left as-is", eventRel, opts.EventName))
		} else {
			out = append(out, File{
				RelPath:   eventRel,
				Content:   merged,
				Overwrite: OverwriteReplace,
			})
		}
	} else {
		body, err := renderEvent("event_types.go.tmpl", data)
		if err != nil {
			return nil, "", err
		}
		out = append(out, File{RelPath: eventRel, Content: body})
	}

	// Optional: scaffold or update the subscriber.
	if opts.Subscriber != "" {
		subRel := path.Join("internal", "features", opts.Subscriber, "subscribers.go")
		subAbs := filepath.Join(opts.ProjectRoot, filepath.FromSlash(subRel))
		if existing, ok := readExisting(subAbs); ok {
			merged, alreadyPresent, err := injectSubscribeCall(existing, data)
			if err != nil {
				return nil, "", fmt.Errorf("inject subscribers.go: %w", err)
			}
			if alreadyPresent {
				notices = append(notices, fmt.Sprintf("  ↻  %s: Subscribe block for %s already present — left as-is", subRel, opts.EventName))
			} else {
				out = append(out, File{
					RelPath:   subRel,
					Content:   merged,
					Overwrite: OverwriteReplace,
				})
			}
		} else {
			body, err := renderEvent("subscribers.go.tmpl", data)
			if err != nil {
				return nil, "", err
			}
			out = append(out, File{RelPath: subRel, Content: body})
		}
	}

	notice := ""
	if len(notices) > 0 {
		notice = "\n" + strings.Join(notices, "\n") + "\n"
	}
	return out, notice, nil
}

// renderEvent loads, parses, executes, and gofmt's one event template.
func renderEvent(name string, data eventData) ([]byte, error) {
	raw, err := fs.ReadFile(templates.Event, "event/"+name)
	if err != nil {
		return nil, err
	}
	t, err := template.New(name).Funcs(template.FuncMap{
		"lower": strings.ToLower,
	}).Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute %s: %w", name, err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("gofmt %s: %w\n--- rendered ---\n%s", name, err, buf.String())
	}
	return formatted, nil
}

// renderEventFragment renders a template without trying to gofmt the output
// (used for fragments that aren't complete Go files on their own).
func renderEventFragment(name string, data eventData) (string, error) {
	raw, err := fs.ReadFile(templates.Event, "event/"+name)
	if err != nil {
		return "", err
	}
	t, err := template.New(name).Funcs(template.FuncMap{
		"lower": strings.ToLower,
	}).Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute %s: %w", name, err)
	}
	return buf.String(), nil
}

// EventNextSteps prints the post-generation reminder block.
func EventNextSteps(opts EventOptions) string {
	var b strings.Builder
	b.WriteString("\n  Next steps:\n")
	fmt.Fprintf(&b, "    1. Publish the event in your feature's service_impl.go:\n")
	fmt.Fprintf(&b, "         s.eventBus.Publish(events.%s, events.%sPayload{ ... })\n",
		opts.EventName, opts.EventName)
	if opts.Subscriber != "" {
		fmt.Fprintf(&b, "    2. Ensure %s.RegisterSubscribers(p) is called in internal/server/subscribers.go\n",
			opts.Subscriber)
	}
	return b.String()
}
