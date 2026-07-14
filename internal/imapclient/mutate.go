package imapclient

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	imapcli "github.com/emersion/go-imap/v2/imapclient"
)

func (c *Client) MarkRead(ctx context.Context, folder string, uid uint32) (MutationResult, error) {
	storeFlags := &imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: true,
		Flags:  []imap.Flag{imap.FlagSeen},
	}

	result, err := withMailbox(ctx, c, folder, func(session *imapcli.Client, resolvedFolder string) (MutationResult, error) {
		if err := session.Store(imap.UIDSetNum(imap.UID(uid)), storeFlags, nil).Close(); err != nil {
			return MutationResult{}, fmt.Errorf("mark message %d as read: %w", uid, err)
		}

		return MutationResult{
			Status: "ok",
			Action: "mark_read",
			UID:    uid,
			Folder: resolvedFolder,
		}, nil
	})
	if err != nil {
		return MutationResult{}, err
	}

	if err := c.appendAuditEntry(auditEntry{
		Timestamp: time.Now().UTC(),
		Action:    result.Action,
		UID:       uid,
		Folder:    result.Folder,
	}); err != nil {
		return MutationResult{}, err
	}

	return result, nil
}

func (c *Client) MoveToFolder(ctx context.Context, folder string, uid uint32, targetFolder string) (MutationResult, error) {
	targetFolder = strings.TrimSpace(targetFolder)
	if targetFolder == "" {
		return MutationResult{}, fmt.Errorf("target folder cannot be blank")
	}

	result, err := withMailbox(ctx, c, folder, func(session *imapcli.Client, resolvedFolder string) (MutationResult, error) {
		if _, err := session.Move(imap.UIDSetNum(imap.UID(uid)), targetFolder).Wait(); err != nil {
			return MutationResult{}, fmt.Errorf("move message %d to %q: %w", uid, targetFolder, err)
		}

		return MutationResult{
			Status:       "ok",
			Action:       "move_to_folder",
			UID:          uid,
			Folder:       resolvedFolder,
			TargetFolder: targetFolder,
		}, nil
	})
	if err != nil {
		return MutationResult{}, err
	}

	if err := c.appendAuditEntry(auditEntry{
		Timestamp:    time.Now().UTC(),
		Action:       result.Action,
		UID:          uid,
		Folder:       result.Folder,
		TargetFolder: result.TargetFolder,
	}); err != nil {
		return MutationResult{}, err
	}

	return result, nil
}
