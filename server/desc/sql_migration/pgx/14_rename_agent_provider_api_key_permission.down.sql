UPDATE "manage_menu"
SET permissions = '[{"code":"v1:manage:agent:getSettings","desc":"读取模型配置"},{"code":"v1:manage:agent:saveSelection","desc":"切换 Provider 和模型"},{"code":"v1:manage:agent:saveDeepSeekApiKey","desc":"保存 DeepSeek API Key"},{"code":"v1:manage:agent:loginApiKey","desc":"API Key 登录"},{"code":"v1:manage:agent:startChatGPTLogin","desc":"ChatGPT 登录"},{"code":"v1:manage:agent:logout","desc":"退出模型账号"},{"code":"v1:manage:agent:restartRuntime","desc":"重启 Runtime"}]',
    update_time = CURRENT_TIMESTAMP
WHERE uuid = 'aa10f1d2-8d73-4ff9-9702-4f7395b7a003';
