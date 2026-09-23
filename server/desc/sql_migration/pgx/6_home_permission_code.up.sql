UPDATE "manage_menu"
SET permissions = REPLACE(
        permissions,
        'v1:manage:agent:homeOverview',
        'v1:home:homeOverview'
    ),
    update_time = CURRENT_TIMESTAMP
WHERE route_name = 'home'
  AND permissions LIKE '%v1:manage:agent:homeOverview%';
