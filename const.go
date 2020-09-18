package discovery

const (
	DefaultTempDir = "discovery-local"
)

const (
	FailBack FailType = 0
	FailFast FailType = 1
)

type FailType int

const (
	RegistryConsul RegistryType = "consul"
	RegistryFile   RegistryType = "file"
)

type RegistryType string

func (rtype RegistryType) IsValid() bool {
	switch rtype {
	case RegistryConsul, RegistryFile:
		return true
	}

	return false
}
