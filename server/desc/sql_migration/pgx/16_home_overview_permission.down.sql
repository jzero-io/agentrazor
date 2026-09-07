UPDATE "manage_menu"
SET permissions = '[]',
    update_time = CURRENT_TIMESTAMP
WHERE route_name = 'home';
