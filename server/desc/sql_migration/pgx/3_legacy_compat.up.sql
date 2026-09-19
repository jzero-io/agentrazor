-- Compatibility placeholder for deployments whose schema_migrations table
-- already records version 3 from a historical migration no longer shipped in
-- this repository. Keep this numbered pair so those deployments can advance
-- to later migrations safely.
SELECT 1;
