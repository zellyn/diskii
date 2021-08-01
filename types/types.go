// Package types holds various types that are needed all over the place. They're
// in their own package to avoid circular dependencies.
package types

// Globals holds flags and configuration that are shared globally.
type Globals struct {
	Debug bool

	DiskOperatorFactories []OperatorFactory
}
