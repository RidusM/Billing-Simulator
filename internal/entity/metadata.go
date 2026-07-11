package entity

type Metadata map[string]string

func NewMetadata() Metadata {
	return Metadata{}
}
