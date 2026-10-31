package module

type Flag struct {
	Name        string
	Description string
}

type FlagModule interface {
	Flags() []Flag
}

func (m *HttpModule) Flags() []Flag {
	return []Flag{
		{Name: "--ssl", Description: "Force the https:// scheme on the target."},
	}
}
