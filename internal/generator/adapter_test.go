package generator

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAdapter_FreshCreate_FullSignatures(t *testing.T) {
	root := t.TempDir()
	opts := AdapterOptions{
		Service:     "payment",
		Provider:    "stripe",
		MethodsSpec: "CreateCharge(req ChargeRequest) (*ChargeResponse, error); RefundCharge(id string) error",
		ModulePath:  "github.com/acme/svc",
		ProjectRoot: root,
	}
	files, _, err := GenerateAdapter(opts)
	if err != nil {
		t.Fatalf("GenerateAdapter: %v", err)
	}
	want := []string{
		"internal/adapters/payment/port.go",
		"internal/adapters/payment/stripe/adapter.go",
		"internal/adapters/payment/stripe/adapter_test.go",
	}
	if len(files) != len(want) {
		t.Fatalf("got %d files, want %d", len(files), len(want))
	}
	for i, w := range want {
		if files[i].RelPath != w {
			t.Errorf("file[%d] = %q, want %q", i, files[i].RelPath, w)
		}
		if files[i].Overwrite != OverwriteFail {
			t.Errorf("file[%d] Overwrite = %v, want Fail", i, files[i].Overwrite)
		}
	}
	// Parse every output as valid Go.
	fset := token.NewFileSet()
	for _, f := range files {
		if _, err := parser.ParseFile(fset, f.RelPath, f.Content, parser.AllErrors); err != nil {
			t.Errorf("%s does not parse: %v\n---\n%s", f.RelPath, err, f.Content)
		}
	}

	// port.go must declare PaymentGateway plus placeholder structs.
	port := string(files[0].Content)
	for _, want := range []string{
		"type PaymentGateway interface",
		"CreateCharge(req ChargeRequest) (*ChargeResponse, error)",
		"RefundCharge(id string) error",
		"type ChargeRequest struct{}",
		"type ChargeResponse struct{}",
	} {
		if !strings.Contains(port, want) {
			t.Errorf("port.go missing %q\n---\n%s", want, port)
		}
	}

	// adapter.go must reference the gateway and panic in each method body.
	ad := string(files[1].Content)
	for _, want := range []string{
		"package stripe",
		"\"github.com/acme/svc/internal/adapters/payment\"",
		"NewAdapter() payment.PaymentGateway",
		"panic(\"not implemented: stripe.CreateCharge\")",
		"panic(\"not implemented: stripe.RefundCharge\")",
	} {
		if !strings.Contains(ad, want) {
			t.Errorf("adapter.go missing %q\n---\n%s", want, ad)
		}
	}
}

func TestGenerateAdapter_NamesOnly_FallbackSignature(t *testing.T) {
	files, _, err := GenerateAdapter(AdapterOptions{
		Service:     "sms",
		Provider:    "twilio",
		MethodsSpec: "Send,SendBulk",
		ModulePath:  "github.com/acme/svc",
		ProjectRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	port := string(files[0].Content)
	for _, want := range []string{
		"Send(ctx context.Context) error",
		"SendBulk(ctx context.Context) error",
	} {
		if !strings.Contains(port, want) {
			t.Errorf("port.go missing %q\n---\n%s", want, port)
		}
	}
	if !strings.Contains(port, `"context"`) {
		t.Errorf("port.go must import context\n---\n%s", port)
	}
}

func TestGenerateAdapter_MergesIntoExistingPort(t *testing.T) {
	root := t.TempDir()
	// Seed an existing port.go with one method.
	portDir := filepath.Join(root, "internal", "adapters", "payment")
	if err := os.MkdirAll(portDir, 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `package payment

import "context"

type PaymentGateway interface {
	Existing(ctx context.Context) error
}
`
	if err := os.WriteFile(filepath.Join(portDir, "port.go"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	files, _, err := GenerateAdapter(AdapterOptions{
		Service:     "payment",
		Provider:    "paypal",
		MethodsSpec: "CreatePayment(amount int64) error; CapturePayment(id string) error",
		ModulePath:  "github.com/acme/svc",
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatalf("GenerateAdapter: %v", err)
	}
	// First file is port.go, must be Replace mode.
	if files[0].Overwrite != OverwriteReplace {
		t.Errorf("port.go Overwrite = %v, want Replace", files[0].Overwrite)
	}
	port := string(files[0].Content)
	for _, want := range []string{
		"Existing(ctx context.Context) error", // preserved
		"CreatePayment(amount int64) error",   // appended
		"CapturePayment(id string) error",     // appended
	} {
		if !strings.Contains(port, want) {
			t.Errorf("merged port.go missing %q\n---\n%s", want, port)
		}
	}

	// adapter.go and adapter_test.go for the new provider must be Fail mode.
	if files[1].Overwrite != OverwriteFail || files[2].Overwrite != OverwriteFail {
		t.Errorf("new provider files must be Overwrite=Fail")
	}
}

func TestGenerateAdapter_MergeSkipsExistingMethods(t *testing.T) {
	root := t.TempDir()
	portDir := filepath.Join(root, "internal", "adapters", "sms")
	if err := os.MkdirAll(portDir, 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `package sms

import "context"

type SmsGateway interface {
	Send(ctx context.Context) error
}
`
	if err := os.WriteFile(filepath.Join(portDir, "port.go"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	files, _, err := GenerateAdapter(AdapterOptions{
		Service:     "sms",
		Provider:    "aws",
		MethodsSpec: "Send,Receive", // Send already exists, Receive is new
		ModulePath:  "github.com/acme/svc",
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	port := string(files[0].Content)
	if strings.Count(port, "Send(ctx context.Context) error") != 1 {
		t.Errorf("Send should appear exactly once in merged port.go:\n%s", port)
	}
	if !strings.Contains(port, "Receive(ctx context.Context) error") {
		t.Errorf("Receive must be appended:\n%s", port)
	}
}

func TestGenerateAdapter_RejectsBadNames(t *testing.T) {
	cases := []AdapterOptions{
		{Service: "", Provider: "stripe", MethodsSpec: "Foo", ModulePath: "x"},
		{Service: "Payment", Provider: "stripe", MethodsSpec: "Foo", ModulePath: "x"},
		{Service: "payment", Provider: "", MethodsSpec: "Foo", ModulePath: "x"},
		{Service: "payment", Provider: "1stripe", MethodsSpec: "Foo", ModulePath: "x"},
		{Service: "func", Provider: "stripe", MethodsSpec: "Foo", ModulePath: "x"}, // keyword
	}
	for i, c := range cases {
		if _, _, err := GenerateAdapter(c); err == nil {
			t.Errorf("case %d: expected error, got nil", i)
		}
	}
}

func TestAdapterNextSteps(t *testing.T) {
	out := AdapterNextSteps(AdapterOptions{Service: "payment", Provider: "stripe"})
	for _, want := range []string{
		"Payment payment.PaymentGateway",
		"p.Payment = stripe.NewAdapter()",
		"Inject p.Payment",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("next-steps missing %q\n---\n%s", want, out)
		}
	}
}

func TestGenerateAdapter_StubSiblings_AppendsStubs(t *testing.T) {
	root := t.TempDir()
	portDir := filepath.Join(root, "internal", "adapters", "payment")
	for _, name := range []string{"stripe", "paypal"} {
		if err := os.MkdirAll(filepath.Join(portDir, name), 0o755); err != nil {
			t.Fatal(err)
		}
		body := []byte("package " + name + "\n\nimport \"github.com/acme/svc/internal/adapters/payment\"\n\ntype " + name + "Adapter struct{}\n\nfunc NewAdapter() payment.PaymentGateway { return &" + name + "Adapter{} }\n\nfunc (a *" + name + "Adapter) Existing() error { return nil }\n")
		if err := os.WriteFile(filepath.Join(portDir, name, "adapter.go"), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(portDir, "port.go"), []byte("package payment\n\ntype PaymentGateway interface {\n\tExisting() error\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, _, err := GenerateAdapter(AdapterOptions{
		Service:      "payment",
		Provider:     "adyen",
		MethodsSpec:  "Existing() error; Refund(id string) error",
		ModulePath:   "github.com/acme/svc",
		ProjectRoot:  root,
		StubSiblings: true,
	})
	if err != nil {
		t.Fatalf("GenerateAdapter: %v", err)
	}

	var portReplaced, stripeStubbed, paypalStubbed bool
	for _, f := range files {
		switch f.RelPath {
		case "internal/adapters/payment/port.go":
			portReplaced = f.Overwrite == OverwriteReplace
		case "internal/adapters/payment/stripe/adapter.go":
			stripeStubbed = f.Overwrite == OverwriteReplace &&
				strings.Contains(string(f.Content), `panic("not implemented: stripe.Refund")`)
		case "internal/adapters/payment/paypal/adapter.go":
			paypalStubbed = f.Overwrite == OverwriteReplace &&
				strings.Contains(string(f.Content), `panic("not implemented: paypal.Refund")`)
		}
	}
	if !portReplaced {
		t.Error("expected port.go to be in Replace mode")
	}
	if !stripeStubbed {
		t.Error("expected stripe/adapter.go to have Refund stub")
	}
	if !paypalStubbed {
		t.Error("expected paypal/adapter.go to have Refund stub")
	}
}

func TestGenerateAdapter_StubSiblings_NoOpWhenNothingAdded(t *testing.T) {
	root := t.TempDir()
	portDir := filepath.Join(root, "internal", "adapters", "sms")
	if err := os.MkdirAll(filepath.Join(portDir, "aws"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(portDir, "port.go"), []byte("package sms\n\nimport \"context\"\n\ntype SmsGateway interface {\n\tSend(ctx context.Context) error\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(portDir, "aws", "adapter.go"), []byte("package aws\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, _, err := GenerateAdapter(AdapterOptions{
		Service:      "sms",
		Provider:     "twilio",
		MethodsSpec:  "Send", // already present
		ModulePath:   "github.com/acme/svc",
		ProjectRoot:  root,
		StubSiblings: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.Contains(f.RelPath, "/aws/") {
			t.Errorf("sibling aws should not be modified, got %s", f.RelPath)
		}
		if f.RelPath == "internal/adapters/sms/port.go" {
			t.Errorf("port.go should not be in file list (no merge), got %+v", f)
		}
	}
}

// TestGenerateAdapter_MergeMissingInterface ensures we fail gracefully when an
// existing port.go has no matching interface (e.g. user renamed it).
func TestGenerateAdapter_MergeMissingInterface(t *testing.T) {
	root := t.TempDir()
	portDir := filepath.Join(root, "internal", "adapters", "payment")
	_ = os.MkdirAll(portDir, 0o755)
	bad := []byte("package payment\n\ntype Renamed interface{}\n")
	_ = os.WriteFile(filepath.Join(portDir, "port.go"), bad, 0o644)

	_, _, err := GenerateAdapter(AdapterOptions{
		Service:     "payment",
		Provider:    "stripe",
		MethodsSpec: "Foo",
		ModulePath:  "github.com/acme/svc",
		ProjectRoot: root,
	})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("PaymentGateway")) {
		t.Errorf("expected error mentioning PaymentGateway, got %v", err)
	}
}
