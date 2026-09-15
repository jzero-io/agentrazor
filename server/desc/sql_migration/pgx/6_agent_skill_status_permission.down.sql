DELETE FROM "casbin_rule"
WHERE p_type = 'p'
  AND v1 = 'v1:manage:agent:setSkillStatus';

DELETE FROM "manage_role_menu"
WHERE menu_uuid = 'f110f1d2-8d73-4ff9-9702-4f7395b7a004';

DELETE FROM "manage_menu"
WHERE uuid = 'f110f1d2-8d73-4ff9-9702-4f7395b7a004';
