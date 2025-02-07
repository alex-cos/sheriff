package nodes

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/alex-cos/sheriff/config"
	"github.com/shirou/gopsutil/v4/process"
)

// ServiceNode represents a single service within a dependency tree. Each node
// holds a command to run, configuration for retries and restart policies,
// and references to its parent and children.
type ServiceNode struct {
	Name        string
	Command     string
	Argument    []string
	MaxRetries  int
	Parent      *ServiceNode
	Children    []*ServiceNode
	RestartDeps bool
	cmd         *exec.Cmd
	logger      *slog.Logger
	mu          sync.Mutex
}

// NewNode creates a standalone ServiceNode with the given name, command, and
// arguments. maxRetries sets the number of start attempts.
func NewNode(name, command string, arguments []string, maxRetries int) *ServiceNode {
	node := &ServiceNode{
		Name:        name,
		Command:     command,
		Argument:    arguments,
		MaxRetries:  3,
		Parent:      nil,
		Children:    []*ServiceNode{},
		RestartDeps: false,
	}
	if maxRetries > 0 {
		node.MaxRetries = maxRetries
	}

	return node
}

// NewServiceNodes builds a service tree from a Config. The root node is a
// virtual parent for all top-level services. It detects cyclic dependencies
// and logs parent reassignments when a service is moved under a dependent.
func NewServiceNodes(logger *slog.Logger, config *config.Config) (*ServiceNode, error) {
	root := NewNode("**root**", "", []string{}, 0)
	root.logger = logger.With(slog.String("name", "**root**"))

	for _, service := range config.Services {
		node := root.Find(service.Name)
		if node == nil {
			node = &ServiceNode{
				Name:   service.Name,
				logger: logger.With(slog.String("name", service.Name)),
			}
		}
		node.Command = service.Command
		node.Argument = service.Argument
		node.MaxRetries = service.MaxRetries
		node.RestartDeps = service.RestartDeps
		if node.MaxRetries <= 0 {
			node.MaxRetries = 3
		}

		for _, dep := range service.DependsOn {
			child := root.Find(dep)
			if child == nil {
				child = NewNode(dep, "", []string{}, 0)
				child.logger = logger.With(slog.String("name", dep))
			}
			if child.Find(node.Name) != nil {
				return root,
					fmt.Errorf("detected cyclic dependency between '%s' and '%s' services", node.Name, child.Name)
			}
			if node.hasAncestor(child.Name) {
				return root,
					fmt.Errorf("detected cyclic dependency between '%s' and '%s' services", node.Name, child.Name)
			}
			if child.Parent != nil {
				child.logger.Warn("reassigning parent",
					slog.String("oldParent", child.Parent.Name),
					slog.String("newParent", node.Name),
				)
			}
			node.Add(child)
		}

		if node.Parent == nil {
			root.Add(node)
		}
	}

	return root, nil
}

func (node *ServiceNode) String() string {
	node.mu.Lock()
	defer node.mu.Unlock()

	return node.string(0)
}

func (node *ServiceNode) GetPID() *int32 {
	node.mu.Lock()
	defer node.mu.Unlock()

	return node.getPID()
}

// Add appends a child node and sets its parent reference.
func (node *ServiceNode) Add(child *ServiceNode) {
	if child == nil {
		return
	}

	node.mu.Lock()
	child.mu.Lock()

	child.Parent = node
	node.Children = append(node.Children, child)

	child.mu.Unlock()
	node.mu.Unlock()
}

// Find searches recursively for a child service by name.
// Returns nil if not found.
func (node *ServiceNode) Find(name string) *ServiceNode {
	node.mu.Lock()
	if node.Name == name {
		node.mu.Unlock()
		return node
	}
	children := make([]*ServiceNode, len(node.Children))
	copy(children, node.Children)
	node.mu.Unlock()

	for _, child := range children {
		found := child.Find(name)
		if found != nil {
			return found
		}
	}

	return nil
}

// Start launches the service and all its children recursively.
// Children are started before their parent to honour dependency ordering.
func (node *ServiceNode) Start(ctx context.Context) error {
	node.mu.Lock()
	defer node.mu.Unlock()

	for _, child := range node.Children {
		err := child.Start(ctx)
		if err != nil {
			return err
		}
	}
	if node.Command != "" {
		return node.start(ctx)
	}

	return nil
}

// Stop terminates the service and all its children. It attempts a graceful
// terminate first and falls back to Kill if the context expires.
func (node *ServiceNode) Stop(ctx context.Context) {
	node.mu.Lock()
	defer node.mu.Unlock()

	select {
	case <-ctx.Done():
		return
	default:
	}

	node.stop(ctx)
	for _, child := range node.Children {
		child.Stop(ctx)
	}
}

// RestartStopped recursively checks all services. Any service whose process
// has exited is restarted. When RestartDeps is enabled, the whole sub-tree is
// stopped and re-started to ensure dependency consistency.
func (node *ServiceNode) RestartStopped(ctx context.Context) {
	node.mu.Lock()
	for _, child := range node.Children {
		child.RestartStopped(ctx)
	}
	if node.Command == "" || node.isRunning() {
		node.mu.Unlock()
		return
	}
	node.logger.Warn("service is not running, restarting")
	if node.RestartDeps {
		node.stop(ctx)
		for _, child := range node.Children {
			child.Stop(ctx)
		}
		for _, child := range node.Children {
			err := child.Start(ctx)
			if err != nil {
				child.logger.Error("failed to restart", slog.String("error", err.Error()))
			}
		}
		err := node.start(ctx)
		if err != nil {
			node.logger.Error("failed to restart", slog.String("error", err.Error()))
		}
	} else {
		err := node.start(ctx)
		if err != nil {
			node.logger.Error("failed to restart", slog.String("error", err.Error()))
		}
	}
	node.mu.Unlock()
}

// Status returns the total number of direct children and how many are running.
func (node *ServiceNode) Status() (int, int) {
	node.mu.Lock()
	children := make([]*ServiceNode, len(node.Children))
	copy(children, node.Children)
	node.mu.Unlock()

	total := 0
	running := 0
	for _, child := range children {
		total++
		if child.IsRunning() {
			running++
		}
		t, r := child.Status()
		total += t
		running += r
	}

	return total, running
}

// IsRunning reports whether the underlying OS process is still alive.
func (node *ServiceNode) IsRunning() bool {
	node.mu.Lock()
	defer node.mu.Unlock()

	return node.isRunning()
}

// Non Exported methods

func (node *ServiceNode) getPID() *int32 {
	if node.cmd == nil {
		return nil
	}
	pid := int32(node.cmd.Process.Pid) // nolint: gosec

	return &pid
}

func (node *ServiceNode) isRunning() bool {
	if node.cmd == nil || node.cmd.Process == nil {
		return false
	}
	pid := int32(node.cmd.Process.Pid) // nolint: gosec
	p, err := process.NewProcess(pid)
	if err != nil {
		return false
	}
	running, err := p.IsRunning()
	if err != nil {
		return false
	}

	return running
}

func (node *ServiceNode) string(level int) string {
	var (
		indent strings.Builder
		str    strings.Builder
	)

	for range level {
		indent.WriteString("  ")
	}
	pid := node.getPID()
	fmt.Fprintf(&str, "%sName: %s\n", indent.String(), node.Name)
	fmt.Fprintf(&str, "%sCommand: %s\n", indent.String(), node.Command)
	fmt.Fprintf(&str, "%sArgument: %s\n", indent.String(), node.Argument)
	if pid != nil {
		fmt.Fprintf(&str, "%sPID: %d\n", indent.String(), *pid)
	}
	fmt.Fprintf(&str, "%sIsRunning: %v\n", indent.String(), node.isRunning())
	if len(node.Children) > 0 {
		str.WriteString(indent.String())
		str.WriteString("Children:\n")
		for _, child := range node.Children {
			str.WriteString(child.string(level + 1))
		}
	}

	return str.String()
}

func (node *ServiceNode) hasAncestor(name string) bool {
	for p := node.Parent; p != nil; p = p.Parent {
		if p.Name == name {
			return true
		}
	}
	return false
}

func (node *ServiceNode) start(ctx context.Context) error {
	if node.isRunning() {
		return nil
	}

	path, err := exec.LookPath(node.Command)
	if err != nil {
		node.logger.Error("command not found", slog.String("command", node.Command))
		return fmt.Errorf("command not found: %s", node.Command)
	}

	var lastErr error
	for i := range node.MaxRetries {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		cmd := exec.CommandContext(ctx, path, node.Argument...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Start()
		if err == nil {
			node.cmd = cmd
			node.logger.Info("service has started successfully")
			return nil
		}

		lastErr = err
		node.logger.Error("failed to start",
			slog.String("command", node.Command),
			slog.Int("attempt", i+1),
			slog.Int("maxRetries", node.MaxRetries),
			slog.String("error", err.Error()),
		)
	}

	return fmt.Errorf("failed to start %s after %d attempts: %w",
		node.Command, node.MaxRetries, lastErr)
}

func (node *ServiceNode) stop(ctx context.Context) {
	if node.Command == "" || !node.isRunning() {
		return
	}
	pid := int32(node.cmd.Process.Pid) // nolint: gosec
	p, err := process.NewProcess(pid)
	if err != nil {
		return
	}

	err = p.Terminate()
	if err != nil {
		node.logger.Warn("failed to terminate", slog.String("error", err.Error()))
		err = p.Kill()
		if err != nil {
			node.logger.Error("failed to kill", slog.String("error", err.Error()))
		}
		return
	}

	done := make(chan struct{})
	go func() {
		err := node.cmd.Wait()
		if err != nil {
			node.logger.Warn("failed to wait", slog.String("error", err.Error()))
		}
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		node.logger.Warn("stop timed out, force killing", slog.String("service", node.Name))
		err = p.Kill()
		if err != nil {
			node.logger.Error("failed to kill", slog.String("error", err.Error()))
		}
		<-done
	}
}
