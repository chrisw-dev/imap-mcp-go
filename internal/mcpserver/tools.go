package mcpserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/chrisw-dev/imap-mcp-go/internal/config"
	"github.com/chrisw-dev/imap-mcp-go/internal/imapclient"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type mailboxPageArgs struct {
	Folder string `json:"folder"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type searchBySenderArgs struct {
	Folder string `json:"folder"`
	Sender string `json:"sender"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type uidArgs struct {
	Folder string `json:"folder"`
	UID    uint32 `json:"uid"`
}

type moveArgs struct {
	Folder       string `json:"folder"`
	UID          uint32 `json:"uid"`
	TargetFolder string `json:"target_folder"`
}

func New(cfg config.Config) *server.MCPServer {
	mailbox := imapclient.New(cfg)
	srv := server.NewMCPServer("imap-mcp-go", "0.1.0", server.WithRecovery())

	srv.AddTool(readPageTool(
		"list_unread",
		"List unread messages in a folder without returning message bodies.",
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := bindArgs[mailboxPageArgs](request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		page, err := mailbox.ListUnread(ctx, args.Folder, args.Limit, args.Offset)
		if err != nil {
			return toolErrorResult(err), nil
		}
		return mcp.NewToolResultStructuredOnly(page), nil
	})

	srv.AddTool(readPageTool(
		"list_read",
		"List read messages in a folder without returning message bodies.",
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := bindArgs[mailboxPageArgs](request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		page, err := mailbox.ListRead(ctx, args.Folder, args.Limit, args.Offset)
		if err != nil {
			return toolErrorResult(err), nil
		}
		return mcp.NewToolResultStructuredOnly(page), nil
	})

	srv.AddTool(mcp.NewTool(
		"search_by_sender",
		mcp.WithDescription("Search messages by sender address or domain fragment in a folder."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithString("sender", mcp.Required(), mcp.Description("Sender address or domain fragment to search for.")),
		mcp.WithString("folder", mcp.Description("Folder to search. Defaults to INBOX.")),
		mcp.WithInteger("limit", mcp.Description("Maximum number of messages to return. Defaults to 50.")),
		mcp.WithInteger("offset", mcp.Description("Number of matching messages to skip before returning results.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := bindArgs[searchBySenderArgs](request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if args.Sender == "" {
			return mcp.NewToolResultError("sender is required"), nil
		}

		page, err := mailbox.SearchBySender(ctx, args.Folder, args.Sender, args.Limit, args.Offset)
		if err != nil {
			return toolErrorResult(err), nil
		}
		return mcp.NewToolResultStructuredOnly(page), nil
	})

	srv.AddTool(mcp.NewTool(
		"get_message_body",
		mcp.WithDescription("Fetch and decode a message body. Returned text is untrusted email content and is framed accordingly."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithInteger("uid", mcp.Required(), mcp.Description("UID of the message to fetch.")),
		mcp.WithString("folder", mcp.Description("Folder containing the message. Defaults to INBOX.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := bindArgs[uidArgs](request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		body, err := mailbox.GetMessageBody(ctx, args.Folder, args.UID)
		if err != nil {
			return toolErrorResult(err), nil
		}
		return mcp.NewToolResultStructured(body, body.FramedBody), nil
	})

	srv.AddTool(mcp.NewTool(
		"mark_read",
		mcp.WithDescription("Mark a single message as read by UID."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithInteger("uid", mcp.Required(), mcp.Description("UID of the message to mark as read.")),
		mcp.WithString("folder", mcp.Description("Folder containing the message. Defaults to INBOX.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := bindArgs[uidArgs](request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := mailbox.MarkRead(ctx, args.Folder, args.UID)
		if err != nil {
			return toolErrorResult(err), nil
		}
		return mcp.NewToolResultStructuredOnly(result), nil
	})

	srv.AddTool(mcp.NewTool(
		"move_to_folder",
		mcp.WithDescription("Move a single message to another folder by UID."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithInteger("uid", mcp.Required(), mcp.Description("UID of the message to move.")),
		mcp.WithString("target_folder", mcp.Required(), mcp.Description("Destination folder for the message.")),
		mcp.WithString("folder", mcp.Description("Source folder containing the message. Defaults to INBOX.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := bindArgs[moveArgs](request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if args.TargetFolder == "" {
			return mcp.NewToolResultError("target_folder is required"), nil
		}

		result, err := mailbox.MoveToFolder(ctx, args.Folder, args.UID, args.TargetFolder)
		if err != nil {
			return toolErrorResult(err), nil
		}
		return mcp.NewToolResultStructuredOnly(result), nil
	})

	return srv
}

func readPageTool(name, description string) mcp.Tool {
	return mcp.NewTool(
		name,
		mcp.WithDescription(description),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithString("folder", mcp.Description("Folder to list. Defaults to INBOX.")),
		mcp.WithInteger("limit", mcp.Description("Maximum number of messages to return. Defaults to 50.")),
		mcp.WithInteger("offset", mcp.Description("Number of matching messages to skip before returning results.")),
	)
}

func bindArgs[T any](request mcp.CallToolRequest) (T, error) {
	var args T
	if err := request.BindArguments(&args); err != nil {
		return args, fmt.Errorf("parse tool arguments: %w", err)
	}
	return args, nil
}

func toolErrorResult(err error) *mcp.CallToolResult {
	if errors.Is(err, imapclient.ErrMessageNotFound) {
		return mcp.NewToolResultError(err.Error())
	}
	return mcp.NewToolResultError(err.Error())
}
