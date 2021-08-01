// Package types holds various types that are needed all over the place. They're
// in their own package to avoid circular dependencies.
package types

// Globals holds flags and configuration that are shared globally.
type Globals struct {
	// Debug level (0 = no debugging, 1 = normal user debugging, 2+ is mostly for me)
	Debug int
	// DiskOperatorFactories holds the current list of registered OperatorFactory instances.
	DiskOperatorFactories []OperatorFactory
}
