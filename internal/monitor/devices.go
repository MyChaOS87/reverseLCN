package monitor

type ShadeState int

const (
	ShadeStateStopped ShadeState = iota
	ShadeStateUp
	ShadeStateDown
)

//nolint:unused
type Device struct {
	name string
}

//nolint:unused
type Light struct {
	Device
	status bool
}

//nolint:unused
type Shade struct {
	Device
	status ShadeState
}

//nolint:unused
type DimabableLight struct {
	Device
	dim byte // 0-255
}
