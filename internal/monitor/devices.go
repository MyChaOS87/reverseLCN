package monitor

type ShadeState int

const (
	ShadeStateStopped ShadeState = iota
	ShadeStateUp
	ShadeStateDown
)

type Device struct {
	name string
}

type Light struct {
	Device
	status bool
}

type Shade struct {
	Device
	status ShadeState
}

type DimabableLight struct {
	Device
	dim byte // 0-255
}
