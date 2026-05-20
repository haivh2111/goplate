package project

import "path"

// FeatureRelDir returns the project-relative directory where a feature's files
// live (always forward-slash even on Windows; used both for filesystem joins
// and for printing).
func FeatureRelDir(name string) string {
	return path.Join("internal", "features", name)
}

// FeatureImportPath composes the absolute Go import path for a feature package.
//
//	FeatureImportPath("github.com/acme/svc", "product")
//	  → "github.com/acme/svc/internal/features/product"
func FeatureImportPath(modulePath, featureName string) string {
	return path.Join(modulePath, "internal", "features", featureName)
}
