package processes

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/process"
)

// ProcessInfo holds a snapshot of a running process.
type ProcessInfo struct {
	Pid       int32
	Name      string
	Path      string
	Arguments []string
}

func (info *ProcessInfo) String() string {
	return fmt.Sprintf("%+v\n", *info)
}

// GetProcesses returns a snapshot of all running processes, excluding those
// whose names match the platform-specific FILTERED list.
func GetProcesses() ([]*ProcessInfo, error) {
	infos := []*ProcessInfo{}

	processes, err := process.Processes()
	if err != nil {
		return infos, err
	}
	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}
		if HasToBeFiltered(name) {
			continue
		}
		exe, err := p.Exe()
		if err != nil {
			exe = ""
		}
		args, err := p.CmdlineSlice()
		if err != nil {
			args = []string{}
		}
		// CmdlineSlice returns [exe, arg1, arg2, ...]; drop the executable path
		args = args[min(1, len(args)):]

		infos = append(infos, &ProcessInfo{
			Pid:       p.Pid,
			Name:      name,
			Path:      exe,
			Arguments: args,
		})
	}

	return infos, nil
}

// HasToBeFiltered checks whether a process name matches the platform-specific
// FILTERED list of system processes to exclude from enumeration results.
func HasToBeFiltered(name string) bool {
	for _, f := range FILTERED {
		if f == name {
			return true
		}
	}
	return false
}
