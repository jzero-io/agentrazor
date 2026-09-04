INSERT INTO "manage_role" (uuid, name, status, code, "desc")
VALUES (
    '9c7f5e1a-2b34-4d68-8f90-a1b2c3d4e5f6',
    '普通 Agent 用户',
    '1',
    'R_AGENT_USER',
    '普通 Agent 用户'
)
ON CONFLICT DO NOTHING;
