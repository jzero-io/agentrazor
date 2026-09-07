UPDATE "manage_menu"
SET permissions = '[{"code":"v1:manage:agent:homeOverview","desc":"读取首页概览"}]',
    update_time = CURRENT_TIMESTAMP
WHERE route_name = 'home';
