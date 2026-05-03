package ports

type Kind string

const (
	KindListener   Kind = "listener"
	KindConnection Kind = "connection"
)

type Entry struct {
	Port          int
	Process       string
	PID           int
	Protocol      string
	State         string
	Details       string
	Kind          Kind
	LocalAddress  string
	RemoteAddress string
	RemotePort    int
}

type ListOptions struct {
	IncludeConnections bool
}
