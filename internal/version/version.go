package version

// Version is the goplate release version. It is overwritten at build time via
// -ldflags "-X github.com/haivh2111/goplate/internal/version.Version=vX.Y.Z".
var Version = "dev"
