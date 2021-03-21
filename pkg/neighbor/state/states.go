package state

type State uint8

const (
	UnknownState State = iota
	Incomplete
	Reachable
	Stale
	Delay
	Probe
	Failed
	NoARP
	Permanent
	None
)

func (s State) String() string {
	switch s {
	case UnknownState:
		return `unknown`
	case Incomplete:
		return `incomplete`
	case Reachable:
		return `reachable`
	case Stale:
		return `stale`
	case Delay:
		return `delay`
	case Probe:
		return `probe`
	case Failed:
		return `failed`
	case NoARP:
		return `noarp`
	case Permanent:
		return `permanent`
	case None:
		return `none`
	default:
		return `???`
	}
}
