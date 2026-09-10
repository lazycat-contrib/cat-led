// Package buildinfo contains application metadata set when building the binary.
package buildinfo

// Version is injected by build.sh. Plain go build/run uses the development label.
var Version = "dev"
