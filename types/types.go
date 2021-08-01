package types

// Globals holds flags and configuration that are shared globally.
type Globals struct {
	Debug bool

	DiskOperatorFactories []OperatorFactory
}
