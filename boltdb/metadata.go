package boltdb

import "github.com/btcboost/copernicus/orm"

type metadata struct {
	orm.MetaData
}

func (metadata *metadata) Create(key []byte) (orm.Bucket, error) {
	return nil, nil
}

func (metadata *metadata) CreateIfNotExists(key []byte) (orm.Bucket, error) {
	return nil, nil
}

func (metadata *metadata) Get(key []byte) orm.Bucket {
	return nil
	
}

func Delete(key []byte) error {
	return nil
}
