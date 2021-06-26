package types

// Globals holds flags and configuration that are shared globally.
type Globals struct {
	Debug  bool
	Order  string //Logical-to-physical sector order
	System string // DOS system used for image

	DiskOperatorFactories []OperatorFactory
}
