//go:build windows

package processes

// FILTERED lists Windows system processes that are excluded from enumeration.
var FILTERED = []string{
	"[System Process]",
	"System",
	"Registry",
	"Secure System",
	"dllhost.exe",
	"svchost.exe",
	"smss.exe",
	"csrss.exe",
	"dwm.exe",
	"wininit.exe",
	"lsass.exe",
	"services.exe",
	"fontdrvhost.exe",
}
