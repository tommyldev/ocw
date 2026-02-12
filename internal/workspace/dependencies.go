package workspace

import (
	"fmt"
	"strings"

	"github.com/tommyzliu/ocw/internal/state"
)

// AddDependency adds a dependency from one instance to another.
// The instance identified by instanceID will depend on dependsOnID.
func (m *Manager) AddDependency(instanceID, dependsOnID string) error {
	if instanceID == dependsOnID {
		return fmt.Errorf("an instance cannot depend on itself")
	}

	st, err := m.store.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	var instance *state.Instance
	var dependsOnExists bool
	for i := range st.Instances {
		if st.Instances[i].ID == instanceID {
			instance = &st.Instances[i]
		}
		if st.Instances[i].ID == dependsOnID {
			dependsOnExists = true
		}
	}

	if instance == nil {
		return fmt.Errorf("instance %q not found", instanceID)
	}
	if !dependsOnExists {
		return fmt.Errorf("dependency instance %q not found", dependsOnID)
	}

	for _, dep := range instance.DependsOn {
		if dep == dependsOnID {
			return fmt.Errorf("dependency already exists: %s depends on %s", instanceID, dependsOnID)
		}
	}

	instance.DependsOn = append(instance.DependsOn, dependsOnID)
	if hasCycle(st.Instances) {
		return fmt.Errorf("adding this dependency would create a circular dependency")
	}

	if err := m.store.Save(st); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// RemoveDependency removes a dependency from one instance to another.
func (m *Manager) RemoveDependency(instanceID, dependsOnID string) error {
	return m.store.UpdateInstance(instanceID, func(inst *state.Instance) {
		filtered := make([]string, 0, len(inst.DependsOn))
		for _, dep := range inst.DependsOn {
			if dep != dependsOnID {
				filtered = append(filtered, dep)
			}
		}
		inst.DependsOn = filtered
	})
}

// CheckDependenciesMerged verifies that all dependencies of an instance have been merged.
// Returns the list of unmerged dependency instance IDs if any exist.
func (m *Manager) CheckDependenciesMerged(instanceID string) ([]string, error) {
	st, err := m.store.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	instanceMap := make(map[string]*state.Instance)
	for i := range st.Instances {
		instanceMap[st.Instances[i].ID] = &st.Instances[i]
	}

	instance, ok := instanceMap[instanceID]
	if !ok {
		return nil, fmt.Errorf("instance %q not found", instanceID)
	}

	var unmerged []string
	for _, depID := range instance.DependsOn {
		dep, exists := instanceMap[depID]
		if !exists {
			continue
		}
		if dep.Status != "merged" && dep.Status != "done" {
			unmerged = append(unmerged, depID)
		}
	}

	return unmerged, nil
}

// TopologicalSort returns instances sorted in dependency order (dependencies first).
// Returns an error if a cycle is detected.
func TopologicalSort(instances []state.Instance) ([]state.Instance, error) {
	if hasCycle(instances) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	idToInst := make(map[string]state.Instance)
	inDegree := make(map[string]int)
	dependents := make(map[string][]string)

	for _, inst := range instances {
		idToInst[inst.ID] = inst
		inDegree[inst.ID] = 0
	}

	for _, inst := range instances {
		for _, depID := range inst.DependsOn {
			if _, exists := idToInst[depID]; exists {
				inDegree[inst.ID]++
				dependents[depID] = append(dependents[depID], inst.ID)
			}
		}
	}

	// Kahn's algorithm
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []state.Instance
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		sorted = append(sorted, idToInst[id])

		for _, depID := range dependents[id] {
			inDegree[depID]--
			if inDegree[depID] == 0 {
				queue = append(queue, depID)
			}
		}
	}

	return sorted, nil
}

// FormatDependencyInfo returns a human-readable dependency string for an instance.
// Returns empty string if the instance has no dependencies.
func FormatDependencyInfo(inst state.Instance, allInstances []state.Instance) string {
	if len(inst.DependsOn) == 0 {
		return ""
	}

	nameMap := make(map[string]string)
	for _, i := range allInstances {
		nameMap[i.ID] = i.Name
	}

	var depNames []string
	for _, depID := range inst.DependsOn {
		if name, ok := nameMap[depID]; ok {
			depNames = append(depNames, name)
		} else {
			depNames = append(depNames, depID+" (deleted)")
		}
	}

	return fmt.Sprintf("depends on %s", strings.Join(depNames, ", "))
}

// hasCycle detects if the dependency graph has a cycle using DFS.
func hasCycle(instances []state.Instance) bool {
	idSet := make(map[string]bool)
	depsMap := make(map[string][]string)

	for _, inst := range instances {
		idSet[inst.ID] = true
		depsMap[inst.ID] = inst.DependsOn
	}

	const (
		white = 0
		gray  = 1
		black = 2
	)

	color := make(map[string]int)
	for id := range idSet {
		color[id] = white
	}

	var dfs func(id string) bool
	dfs = func(id string) bool {
		color[id] = gray
		for _, dep := range depsMap[id] {
			if !idSet[dep] {
				continue
			}
			if color[dep] == gray {
				return true
			}
			if color[dep] == white {
				if dfs(dep) {
					return true
				}
			}
		}
		color[id] = black
		return false
	}

	for id := range idSet {
		if color[id] == white {
			if dfs(id) {
				return true
			}
		}
	}

	return false
}
