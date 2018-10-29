package mapcontainer

type Lesser interface {
	Less(other Lesser) bool
}

type MapContainer interface {
	Insert(Lesser) bool
	Search(Lesser) (Lesser, bool)
	ReplaceOrInsert(Lesser) Lesser
}

type MapContainerIterator interface {
	Next() (Lesser, bool)
}
