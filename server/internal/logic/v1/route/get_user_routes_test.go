package route

import (
	"testing"

	"github.com/jzero-io/agentrazor/server/internal/model/manage_menu"
)

func TestConvertPluginIframeRouteStaysInAdminLayout(t *testing.T) {
	const (
		parentUUID = "plugin-management"
		childUUID  = "discord-plugin"
		pageURL    = "/plugins/discord_chat_bot/admin/"
	)

	menus := []*manage_menu.ManageMenu{
		{
			Uuid:      parentUUID,
			MenuName:  "插件管理",
			RouteName: "plugin_management",
			RoutePath: "/plugins",
			Component: "layout.base",
		},
		{
			Uuid:       childUUID,
			ParentUuid: parentUUID,
			MenuName:   "Discord 支持机器人",
			RouteName:  "plugin_discord_chat_bot",
			RoutePath:  "/plugins/discord-chat-bot",
			Component:  "view.iframe-page",
			Href:       pageURL,
		},
	}

	_, routesByUUID := convert(menus)
	child := routesByUUID[childUUID]
	if child == nil {
		t.Fatal("plugin child route is missing")
	}
	if child.Meta.Href != "" {
		t.Fatalf("plugin iframe route must not be an external href, got %q", child.Meta.Href)
	}
	props, ok := child.Props.(map[string]any)
	if !ok {
		t.Fatalf("plugin iframe props type = %T, want map[string]any", child.Props)
	}
	if got := props["url"]; got != pageURL {
		t.Fatalf("plugin iframe url = %v, want %q", got, pageURL)
	}

	tree := buildRouteTree(menus, routesByUUID, "")
	if len(tree) != 1 || tree[0].Component != "layout.base" {
		t.Fatalf("plugin parent must be an Admin layout route, got %#v", tree)
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].Name != "plugin_discord_chat_bot" {
		t.Fatalf("plugin child must remain nested under the Admin layout, got %#v", tree[0].Children)
	}
}
