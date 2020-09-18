package dumper

const (
	FormatConsul    FormatType = "consul"
	FormatDiscovery FormatType = "discovery"
)

type FormatType string

func (ftype FormatType) IsValid() bool {
	switch ftype {
	case FormatConsul, FormatDiscovery:
		return true
	}

	return false
}
