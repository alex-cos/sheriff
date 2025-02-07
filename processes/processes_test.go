package processes_test

import (
	"testing"

	"github.com/alex-cos/sheriff/processes"
	"github.com/stretchr/testify/assert"
)

func TestHasToBeFiltered(t *testing.T) {
	t.Parallel()

	t.Run("all FILTERED entries return true", func(t *testing.T) {
		t.Parallel()

		for _, name := range processes.FILTERED {
			assert.True(t, processes.HasToBeFiltered(name), "expected %q to be filtered", name)
		}
	})

	t.Run("normal processes are not filtered", func(t *testing.T) {
		t.Parallel()

		notFiltered := []string{
			"myapp",
			"nginx",
			"postgres",
			"",
		}
		for _, name := range notFiltered {
			assert.False(t, processes.HasToBeFiltered(name), "expected %q not to be filtered", name)
		}
	})
}

func TestGetProcesses(t *testing.T) {
	t.Parallel()

	list, err := processes.GetProcesses()
	assert.NoError(t, err)
	assert.NotEmpty(t, list, "should return at least the current process")

	for _, p := range list {
		assert.GreaterOrEqual(t, p.Pid, int32(0), "PID must be non-negative")
		assert.NotEmpty(t, p.Name, "process name must not be empty")
		assert.False(t, processes.HasToBeFiltered(p.Name), "filtered process should not appear: %s", p.Name)
	}
}

func TestProcessInfo_String(t *testing.T) {
	t.Parallel()

	info := &processes.ProcessInfo{
		Pid:       1234,
		Name:      "test.exe",
		Path:      "C:\\test\\test.exe",
		Arguments: []string{"-v"},
	}

	str := info.String()
	assert.Contains(t, str, "1234")
	assert.Contains(t, str, "test.exe")
}
