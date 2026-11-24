INSERT INTO "manage_menu" (
  uuid, create_time, update_time, status, parent_uuid, menu_type, menu_name,
  hide_in_menu, active_menu, "order", route_name, route_path, component,
  icon, icon_type, i18n_key, keep_alive, href, multi_tab, fixed_index_in_tab,
  query, permissions, constant, button_code
) VALUES (
  '93d5b6d9-4ab6-46d2-b9b6-ecad24cd0001', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
  '1', '7d4c5bf1-4e92-4b50-95f1-0e166b90d001', '2', 'Telegram 问答机器人',
  0, '', 2, 'plugin_telegram_chat_bot', '/plugins/telegram-chat-bot', 'view.iframe-page',
  'mdi:telegram', '1', 'route.plugin_telegram_chat_bot', 0,
  '/plugins/telegram_chat_bot/admin/', 0, 0, '[]',
  '[{"code":"v1:plugin:telegramChatBot:getStatus","desc":"读取 Telegram 机器人状态"},{"code":"v1:plugin:telegramChatBot:getConfiguration","desc":"查看 Telegram 配置"},{"code":"v1:plugin:telegramChatBot:saveConfiguration","desc":"保存 Telegram 配置"}]',
  0, ''
)
ON CONFLICT (uuid) DO UPDATE SET
  parent_uuid = EXCLUDED.parent_uuid,
  menu_type = EXCLUDED.menu_type,
  menu_name = EXCLUDED.menu_name,
  "order" = EXCLUDED."order",
  route_name = EXCLUDED.route_name,
  route_path = EXCLUDED.route_path,
  component = EXCLUDED.component,
  i18n_key = EXCLUDED.i18n_key,
  href = EXCLUDED.href,
  permissions = EXCLUDED.permissions,
  update_time = CURRENT_TIMESTAMP;

INSERT INTO "manage_role_menu" (uuid, create_time, update_time, role_uuid, menu_uuid, is_home)
SELECT md5(role.uuid || ':93d5b6d9-4ab6-46d2-b9b6-ecad24cd0001'),
       CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, role.uuid,
       '93d5b6d9-4ab6-46d2-b9b6-ecad24cd0001', 0
FROM "manage_role" AS role
WHERE role.code = 'R_SUPER'
ON CONFLICT (uuid) DO NOTHING;
