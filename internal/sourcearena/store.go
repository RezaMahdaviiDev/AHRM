package sourcearena

import "context"

type NopRawStore struct{}

func (NopRawStore) SaveRaw(context.Context, string, int, []byte) error { return nil }
