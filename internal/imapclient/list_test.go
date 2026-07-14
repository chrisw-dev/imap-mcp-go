package imapclient

import (
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestPaginateUIDs(t *testing.T) {
	uids := []imap.UID{5, 4, 3, 2, 1}

	page := paginateUIDs(uids, 2, 1)
	if len(page) != 2 {
		t.Fatalf("len(page) = %d, want 2", len(page))
	}
	if page[0] != 4 || page[1] != 3 {
		t.Fatalf("page = %v, want [4 3]", page)
	}

	page = paginateUIDs(uids, 2, 99)
	if len(page) != 0 {
		t.Fatalf("len(page) = %d, want 0", len(page))
	}
}

func TestReverseUIDs(t *testing.T) {
	uids := []imap.UID{1, 2, 3}
	reverseUIDs(uids)

	if uids[0] != 3 || uids[1] != 2 || uids[2] != 1 {
		t.Fatalf("uids = %v, want [3 2 1]", uids)
	}
}
