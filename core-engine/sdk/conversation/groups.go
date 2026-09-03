package conversation

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"
)

// ListGroups lists conversation groups owned by the authenticated account.
func (c *Client) ListGroups(ctx context.Context) ([]Group, error) {
	var response struct {
		Groups []Group `json:"groups"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/conversation-groups", nil, &response); err != nil {
		return nil, err
	}
	return response.Groups, nil
}

// CreateGroup creates a conversation group.
func (c *Client) CreateGroup(ctx context.Context, name string) (*Group, error) {
	name, err := validGroupName(name)
	if err != nil {
		return nil, err
	}
	var response Group
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/conversation-groups", map[string]string{"name": name}, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// UpdateGroup renames a conversation group.
func (c *Client) UpdateGroup(ctx context.Context, groupID, name string) (*Group, error) {
	path, err := groupPath(groupID, "")
	if err != nil {
		return nil, err
	}
	name, err = validGroupName(name)
	if err != nil {
		return nil, err
	}
	var response Group
	if err := c.doJSON(ctx, http.MethodPatch, path, map[string]string{"name": name}, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// DeleteGroup deletes a group and moves its conversations back to the
// ungrouped conversation list.
func (c *Client) DeleteGroup(ctx context.Context, groupID string) error {
	path, err := groupPath(groupID, "")
	if err != nil {
		return err
	}
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil)
}

// ArchiveGroupConversations archives every conversation in a group.
func (c *Client) ArchiveGroupConversations(ctx context.Context, groupID string) error {
	path, err := groupPath(groupID, "/archive-conversations")
	if err != nil {
		return err
	}
	return c.doJSON(ctx, http.MethodPost, path, nil, nil)
}

// DeleteGroupArchivedConversations permanently deletes all archived
// conversations in a group.
func (c *Client) DeleteGroupArchivedConversations(ctx context.Context, groupID string) error {
	path, err := groupPath(groupID, "/delete-archived-conversations")
	if err != nil {
		return err
	}
	return c.doJSON(ctx, http.MethodPost, path, nil, nil)
}

func validGroupName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("conversation SDK: group name is required")
	}
	if utf8.RuneCountInString(name) > 40 {
		return "", errors.New("conversation SDK: group name must not exceed 40 characters")
	}
	return name, nil
}

func groupPath(groupID, suffix string) (string, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return "", errors.New("conversation SDK: group ID is required")
	}
	return "/api/v1/conversation-groups/" + url.PathEscape(groupID) + suffix, nil
}
