//go:build !windows

package processes

// FILTERED lists Unix system processes that are excluded from enumeration.
var FILTERED = []string{
	"init",
	"systemd",
	"kthreadd",
}
