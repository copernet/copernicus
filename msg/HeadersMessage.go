package msg

import "copernicus/model"

type HeadersMessage struct {
	Blocks []*model.Block
}
