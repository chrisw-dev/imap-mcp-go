package imapclient

import (
	"context"
	"fmt"

	"github.com/emersion/go-imap/v2"
	imapcli "github.com/emersion/go-imap/v2/imapclient"
)

func (c *Client) ListUnread(ctx context.Context, folder string, limit, offset int) (MessagePage, error) {
	return c.listByCriteria(ctx, folder, limit, offset, &imap.SearchCriteria{
		NotFlag: []imap.Flag{imap.FlagSeen},
	})
}

func (c *Client) ListRead(ctx context.Context, folder string, limit, offset int) (MessagePage, error) {
	return c.listByCriteria(ctx, folder, limit, offset, &imap.SearchCriteria{
		Flag: []imap.Flag{imap.FlagSeen},
	})
}

func (c *Client) listByCriteria(ctx context.Context, folder string, limit, offset int, criteria *imap.SearchCriteria) (MessagePage, error) {
	resolvedLimit, resolvedOffset, err := c.cfg.ResolvePagination(limit, offset)
	if err != nil {
		return MessagePage{}, err
	}

	return withMailbox(ctx, c, folder, func(session *imapcli.Client, resolvedFolder string) (MessagePage, error) {
		searchData, err := session.UIDSearch(criteria, &imap.SearchOptions{ReturnAll: true}).Wait()
		if err != nil {
			return MessagePage{}, fmt.Errorf("search messages: %w", err)
		}

		uids := searchData.AllUIDs()
		reverseUIDs(uids)

		pageUIDs := paginateUIDs(uids, resolvedLimit, resolvedOffset)
		messages, err := fetchSummaries(session, resolvedFolder, pageUIDs)
		if err != nil {
			return MessagePage{}, err
		}

		return MessagePage{
			Folder:   resolvedFolder,
			Limit:    resolvedLimit,
			Offset:   resolvedOffset,
			Total:    len(uids),
			Messages: messages,
		}, nil
	})
}

func fetchSummaries(session *imapcli.Client, folder string, uids []imap.UID) ([]MessageSummary, error) {
	if len(uids) == 0 {
		return []MessageSummary{}, nil
	}

	fetchOptions := &imap.FetchOptions{
		UID:        true,
		Envelope:   true,
		RFC822Size: true,
	}

	buffers, err := session.Fetch(imap.UIDSetNum(uids...), fetchOptions).Collect()
	if err != nil {
		return nil, fmt.Errorf("fetch message summaries: %w", err)
	}

	byUID := make(map[imap.UID]*imapcli.FetchMessageBuffer, len(buffers))
	for _, buffer := range buffers {
		byUID[buffer.UID] = buffer
	}

	summaries := make([]MessageSummary, 0, len(uids))
	for _, uid := range uids {
		buffer, ok := byUID[uid]
		if !ok {
			continue
		}

		summaries = append(summaries, MessageSummary{
			UID:     uint32(buffer.UID),
			Folder:  folder,
			Sender:  formatSender(buffer.Envelope),
			Subject: envelopeSubject(buffer.Envelope),
			Date:    envelopeDate(buffer.Envelope),
			Size:    buffer.RFC822Size,
		})
	}

	return summaries, nil
}
