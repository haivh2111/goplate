package generator

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateEvent_FreshCreate(t *testing.T) {
	root := t.TempDir()
	files, notice, err := GenerateEvent(EventOptions{
		EventName:   "OrderPlaced",
		PayloadSpec: "OrderID:uint,UserID:uint,TotalAmount:float64",
		Subscriber:  "notification",
		ModulePath:  "github.com/acme/svc",
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatalf("GenerateEvent: %v", err)
	}
	if notice != "" {
		t.Errorf("expected empty notice on fresh run, got %q", notice)
	}
	want := []string{
		"internal/events/event_types.go",
		"internal/features/notification/subscribers.go",
	}
	if len(files) != len(want) {
		t.Fatalf("got %d files, want %d", len(files), len(want))
	}
	for i, w := range want {
		if files[i].RelPath != w {
			t.Errorf("file[%d] = %q, want %q", i, files[i].RelPath, w)
		}
	}
	fset := token.NewFileSet()
	for _, f := range files {
		if _, err := parser.ParseFile(fset, f.RelPath, f.Content, parser.AllErrors); err != nil {
			t.Errorf("%s does not parse:\n%v\n---\n%s", f.RelPath, err, f.Content)
		}
	}
	et := string(files[0].Content)
	for _, want := range []string{
		`const OrderPlaced = "order.placed"`,
		"type OrderPlacedPayload struct",
		"OrderID     uint",
		"TotalAmount float64",
	} {
		if !strings.Contains(et, want) {
			t.Errorf("event_types.go missing %q\n---\n%s", want, et)
		}
	}
	sub := string(files[1].Content)
	for _, want := range []string{
		"package notification",
		"p.EventBus.Subscribe(events.OrderPlaced",
		"events.OrderPlacedPayload",
	} {
		if !strings.Contains(sub, want) {
			t.Errorf("subscribers.go missing %q\n---\n%s", want, sub)
		}
	}
}

func TestGenerateEvent_AppendsToExistingEventTypes(t *testing.T) {
	root := t.TempDir()
	evtDir := filepath.Join(root, "internal", "events")
	if err := os.MkdirAll(evtDir, 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `package events

const OrderPlaced = "order.placed"

type OrderPlacedPayload struct {
	OrderID uint
}
`
	if err := os.WriteFile(filepath.Join(evtDir, "event_types.go"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	files, _, err := GenerateEvent(EventOptions{
		EventName:   "PaymentFailed",
		PayloadSpec: "OrderID:uint,Reason:string",
		ModulePath:  "github.com/acme/svc",
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Overwrite != OverwriteReplace {
		t.Fatalf("expected 1 file in Replace mode, got %+v", files)
	}
	merged := string(files[0].Content)
	for _, want := range []string{
		`const OrderPlaced = "order.placed"`,    // preserved
		`const PaymentFailed = "payment.failed"`, // appended
		"type PaymentFailedPayload struct",
		"OrderID uint",
		"Reason  string",
	} {
		if !strings.Contains(merged, want) {
			t.Errorf("merged event_types.go missing %q\n---\n%s", want, merged)
		}
	}
}

func TestGenerateEvent_IdempotentOnReRun(t *testing.T) {
	root := t.TempDir()
	evtDir := filepath.Join(root, "internal", "events")
	subDir := filepath.Join(root, "internal", "features", "notification")
	if err := os.MkdirAll(evtDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(evtDir, "event_types.go"), []byte(`package events

const OrderPlaced = "order.placed"

type OrderPlacedPayload struct {
	OrderID uint
}
`), 0o644)
	_ = os.WriteFile(filepath.Join(subDir, "subscribers.go"), []byte(`package notification

import (
	"github.com/acme/svc/internal/events"
	"github.com/acme/svc/internal/server"
)

func RegisterSubscribers(p *server.Providers) {
	p.EventBus.Subscribe(events.OrderPlaced, func(payload any) {})
}
`), 0o644)

	files, notice, err := GenerateEvent(EventOptions{
		EventName:   "OrderPlaced",
		PayloadSpec: "OrderID:uint",
		Subscriber:  "notification",
		ModulePath:  "github.com/acme/svc",
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("expected no file changes on idempotent re-run, got %d", len(files))
	}
	if !strings.Contains(notice, "already declared") {
		t.Errorf("notice should mention already-declared, got %q", notice)
	}
}

func TestGenerateEvent_InjectsIntoExistingSubscribers(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "internal", "features", "notification")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `package notification

import (
	"github.com/acme/svc/internal/events"
	"github.com/acme/svc/internal/server"
)

func RegisterSubscribers(p *server.Providers) {
	p.EventBus.Subscribe(events.OrderPlaced, func(payload any) {})
}
`
	if err := os.WriteFile(filepath.Join(subDir, "subscribers.go"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	files, _, err := GenerateEvent(EventOptions{
		EventName:   "PaymentFailed",
		PayloadSpec: "Reason:string",
		Subscriber:  "notification",
		ModulePath:  "github.com/acme/svc",
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	var sub string
	for _, f := range files {
		if strings.HasSuffix(f.RelPath, "subscribers.go") {
			if f.Overwrite != OverwriteReplace {
				t.Errorf("subscribers.go should be Replace mode")
			}
			sub = string(f.Content)
		}
	}
	if !strings.Contains(sub, "events.OrderPlaced") {
		t.Errorf("existing Subscribe must be preserved\n---\n%s", sub)
	}
	if !strings.Contains(sub, "events.PaymentFailed") {
		t.Errorf("new Subscribe must be injected\n---\n%s", sub)
	}
}

func TestGenerateEvent_NoSubscriberFlag(t *testing.T) {
	files, _, err := GenerateEvent(EventOptions{
		EventName:   "Ping",
		PayloadSpec: "",
		ModulePath:  "github.com/acme/svc",
		ProjectRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file (only event_types.go), got %d", len(files))
	}
}

func TestGenerateEvent_RejectsBadNames(t *testing.T) {
	cases := []EventOptions{
		{EventName: "", ModulePath: "x", ProjectRoot: t.TempDir()},
		{EventName: "orderPlaced", ModulePath: "x", ProjectRoot: t.TempDir()},
		{EventName: "Order_Placed", ModulePath: "x", ProjectRoot: t.TempDir()},
	}
	for i, c := range cases {
		if _, _, err := GenerateEvent(c); err == nil {
			t.Errorf("case %d: expected error, got nil", i)
		}
	}
}

func TestEventNextSteps(t *testing.T) {
	out := EventNextSteps(EventOptions{EventName: "OrderPlaced", Subscriber: "notification"})
	for _, want := range []string{
		"s.eventBus.Publish(events.OrderPlaced",
		"notification.RegisterSubscribers(p)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("next-steps missing %q\n---\n%s", want, out)
		}
	}
}
