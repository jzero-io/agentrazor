package menu

import (
	"testing"

	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/menu"
)

func TestBuildSimpleMenuTreePreservesI18nKey(t *testing.T) {
	menus := []*types.SystemMenu{
		{
			Uuid:       "parent",
			MenuName:   "系统管理",
			I18nKey:    "route.manage",
			ParentUuid: "",
		},
		{
			Uuid:       "child",
			MenuName:   "用户管理",
			I18nKey:    "route.manage_user",
			ParentUuid: "parent",
		},
	}

	tree := buildSimpleMenuTree(menus, "")
	if len(tree) != 1 {
		t.Fatalf("expected one root node, got %d", len(tree))
	}
	if tree[0].I18nKey != "route.manage" {
		t.Fatalf("expected root i18n key %q, got %q", "route.manage", tree[0].I18nKey)
	}
	if len(tree[0].Children) != 1 {
		t.Fatalf("expected one child node, got %d", len(tree[0].Children))
	}
	if tree[0].Children[0].I18nKey != "route.manage_user" {
		t.Fatalf("expected child i18n key %q, got %q", "route.manage_user", tree[0].Children[0].I18nKey)
	}
}
