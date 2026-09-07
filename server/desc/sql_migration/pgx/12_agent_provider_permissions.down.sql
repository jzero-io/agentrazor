UPDATE "manage_menu"
SET permissions = '[{"code":"v1:manage:agent:listConfigFiles","desc":"配置文件列表"},{"code":"v1:manage:agent:configFile","desc":"读取配置文件"},{"code":"v1:manage:agent:updateConfigFile","desc":"更新配置文件"},{"code":"v1:manage:agent:runtimeStatus","desc":"Runtime 状态"},{"code":"v1:manage:agent:restartRuntime","desc":"重启 Runtime"}]',
    update_time = CURRENT_TIMESTAMP
WHERE uuid = 'aa10f1d2-8d73-4ff9-9702-4f7395b7a003';
