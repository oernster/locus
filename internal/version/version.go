// Package version carries the application version. The value is supplied by the
// build from the VERSION file at the repository root, which is the single place
// a real version string is written down.
package version

// Version is the application version string.
//
// It is deliberately a var rather than a const. The build overwrites it with the
// contents of VERSION through -ldflags "-X". A -X flag aimed at a const is
// silently ignored: the binary then ships the sentinel below with no error
// anywhere to read, so this must never be tightened to a const.
//
// The sentinel is what an unstamped build reports. Anyone running go build or go
// test directly sees it and knows the figure did not come from VERSION.
var Version = "0.0.0-dev"
