package mapcontainer

type Lesser interface {
	Less(other Lesser) bool
}

type MapContainer interface {
	//Insert(Lesser) bool
	Search(Lesser) (Lesser, bool)
	ReplaceOrInsert(Lesser) Lesser
	Delete(Lesser) (deleted Lesser, found bool)
	Min() (less Lesser, found bool)
	Ascend(iterHandler func(i Lesser) bool)

	Len() int
	Max() (less Lesser, found bool)
	DeleteMax() (less Lesser, found bool)
}
