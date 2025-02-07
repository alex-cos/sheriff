package nodes_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/alex-cos/sheriff/config"
	"github.com/alex-cos/sheriff/nodes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func TestNewNode(t *testing.T) {
	t.Parallel()

	node := nodes.NewNode("test", "echo", []string{"hello"}, 3)

	assert.Equal(t, "test", node.Name)
	assert.Equal(t, "echo", node.Command)
	assert.Equal(t, []string{"hello"}, node.Argument)
	assert.Nil(t, node.Parent)
	assert.Empty(t, node.Children)
}

func TestServiceNode_Add(t *testing.T) {
	t.Parallel()

	parent := nodes.NewNode("parent", "", nil, 3)
	child := nodes.NewNode("child", "", nil, 3)

	parent.Add(child)

	assert.Equal(t, parent, child.Parent)
	require.Len(t, parent.Children, 1)
	assert.Equal(t, "child", parent.Children[0].Name)
}

func TestServiceNode_Add_NilChild(t *testing.T) {
	t.Parallel()

	parent := nodes.NewNode("parent", "", nil, 3)
	parent.Add(nil)

	assert.Empty(t, parent.Children)
}

func TestServiceNode_Find(t *testing.T) {
	t.Parallel()

	root := nodes.NewNode("root", "", nil, 3)
	child := nodes.NewNode("child", "", nil, 3)
	grandchild := nodes.NewNode("grandchild", "", nil, 3)
	root.Add(child)
	child.Add(grandchild)

	t.Run("find root", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, root, root.Find("root"))
	})

	t.Run("find child", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, child, root.Find("child"))
	})

	t.Run("find grandchild recursive", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, grandchild, root.Find("grandchild"))
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, root.Find("nonexistent"))
	})
}

func TestNewServiceNodes_NoDependencies(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Services: []config.ServiceConfig{
			{Name: "svc1", Command: "echo", Argument: []string{"a"}},
			{Name: "svc2", Command: "echo", Argument: []string{"b"}},
		},
	}

	root, err := nodes.NewServiceNodes(testLogger(), cfg)
	require.NoError(t, err)

	assert.Equal(t, "**root**", root.Name)
	require.Len(t, root.Children, 2)
	assert.Equal(t, "svc1", root.Children[0].Name)
	assert.Equal(t, "svc2", root.Children[1].Name)
}

func TestNewServiceNodes_WithDependencies(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Services: []config.ServiceConfig{
			{Name: "app", Command: "echo", DependsOn: []string{"db"}},
			{Name: "db", Command: "echo"},
		},
	}

	root, err := nodes.NewServiceNodes(testLogger(), cfg)
	require.NoError(t, err)

	app := root.Find("app")
	db := root.Find("db")
	require.NotNil(t, app)
	require.NotNil(t, db)

	assert.Equal(t, app, db.Parent, "db should be child of app (app depends on db)")
	assert.Contains(t, app.Children, db)
}

func TestNewServiceNodes_CycleDirect(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Services: []config.ServiceConfig{
			{Name: "A", DependsOn: []string{"B"}},
			{Name: "B", DependsOn: []string{"A"}},
		},
	}

	_, err := nodes.NewServiceNodes(testLogger(), cfg)
	require.Error(t, err)
	assert.ErrorContains(t, err, "cyclic dependency")
}

func TestNewServiceNodes_CycleIndirect(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Services: []config.ServiceConfig{
			{Name: "A", DependsOn: []string{"B"}},
			{Name: "B", DependsOn: []string{"C"}},
			{Name: "C", DependsOn: []string{"A"}},
		},
	}

	_, err := nodes.NewServiceNodes(testLogger(), cfg)
	require.Error(t, err)
	assert.ErrorContains(t, err, "cyclic dependency")
}

func TestNewServiceNodes_CycleSelfReferencing(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Services: []config.ServiceConfig{
			{Name: "A", DependsOn: []string{"A"}},
		},
	}

	_, err := nodes.NewServiceNodes(testLogger(), cfg)
	require.Error(t, err)
	assert.ErrorContains(t, err, "cyclic dependency")
}

func TestNewServiceNodes_MultipleDependents(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Services: []config.ServiceConfig{
			{Name: "shared", Command: "echo"},
			{Name: "app1", Command: "echo", DependsOn: []string{"shared"}},
			{Name: "app2", Command: "echo", DependsOn: []string{"shared"}},
		},
	}

	root, err := nodes.NewServiceNodes(testLogger(), cfg)
	require.NoError(t, err)

	shared := root.Find("shared")
	require.NotNil(t, shared)

	assert.Equal(t, "app2", shared.Parent.Name, "last dependent wins as parent")
}

func TestNewServiceNodes_UnknownDependency(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Services: []config.ServiceConfig{
			{Name: "app", DependsOn: []string{"nonexistent"}},
		},
	}

	root, err := nodes.NewServiceNodes(testLogger(), cfg)
	require.NoError(t, err)

	child := root.Find("nonexistent")
	require.NotNil(t, child)
	assert.Equal(t, "app", child.Parent.Name)
}

func TestServiceNode_String(t *testing.T) {
	t.Parallel()

	root := nodes.NewNode("**root**", "", nil, 3)
	child := nodes.NewNode("child", "echo", []string{"hello"}, 3)
	root.Add(child)

	str := root.String()
	assert.Contains(t, str, "**root**")
	assert.Contains(t, str, "child")
	assert.Contains(t, str, "echo")
	assert.Contains(t, str, "hello")
	assert.Contains(t, str, "Children:")
}
