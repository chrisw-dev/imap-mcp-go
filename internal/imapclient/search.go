package imapclient

import (
	"context"

	"github.com/emersion/go-imap/v2"
)

func (c *Client) SearchBySender(ctx context.Context, folder, sender string, limit, offset int) (MessagePage, error) {
	criteria := &imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{
			{Key: "From", Value: sender},
		},
	}

	return c.listByCriteria(ctx, folder, limit, offset, criteria)
}
