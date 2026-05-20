package naming

import "testing"

func TestToPascal(t *testing.T) {
	cases := map[string]string{
		"":            "",
		"product":     "Product",
		"order_item":  "OrderItem",
		"order-item":  "OrderItem",
		"user_id":     "UserID",
		"api_url":     "APIURL",
		"json_data":   "JSONData",
		"PRODUCT":     "PRODUCT", // already-cased input is preserved
		"occurredAt":  "OccurredAt",
		"orderID":     "OrderID",
	}
	for in, want := range cases {
		if got := ToPascal(in); got != want {
			t.Errorf("ToPascal(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestToCamel(t *testing.T) {
	cases := map[string]string{
		"":           "",
		"product":    "product",
		"order_item": "orderItem",
		"user_id":    "userID",
	}
	for in, want := range cases {
		if got := ToCamel(in); got != want {
			t.Errorf("ToCamel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestToDotted(t *testing.T) {
	cases := map[string]string{
		"":                       "",
		"OrderPlaced":            "order.placed",
		"PaymentFailed":          "payment.failed",
		"UserRegistered":         "user.registered",
		"OrderShippedToWarehouse": "order.shipped.to.warehouse",
		"APIKeyRotated":          "api.key.rotated",
		"X":                      "x",
		"HTTPRequest":            "http.request",
	}
	for in, want := range cases {
		if got := ToDotted(in); got != want {
			t.Errorf("ToDotted(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPluralize(t *testing.T) {
	cases := map[string]string{
		"product":  "products",
		"order":    "orders",
		"category": "categories",
		"bus":      "buses",
		"box":      "boxes",
		"buzz":     "buzzes",
		"match":    "matches",
		"dish":     "dishes",
		"day":      "days", // vowel before y
		"boy":      "boys",
		"":         "",
	}
	for in, want := range cases {
		if got := Pluralize(in); got != want {
			t.Errorf("Pluralize(%q) = %q, want %q", in, got, want)
		}
	}
}
