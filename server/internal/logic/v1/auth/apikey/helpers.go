package apikey

import (
	agentapikeymodel "github.com/jzero-io/agentrazor/server/internal/model/agent_api_key"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/auth/apikey"
)

func keyHint(key string) string {
	if len(key) <= 15 {
		return key
	}
	return key[:11] + "…" + key[len(key)-4:]
}

func toAPIKey(row *agentapikeymodel.AgentApiKey) types.ApiKey {
	return types.ApiKey{
		Id:        row.Uuid,
		KeyHint:   row.KeyHint,
		CreatedAt: row.CreateTime.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	}
}
