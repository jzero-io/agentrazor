package pluginconfiguration

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type Record struct {
	EncryptedValue string    `db:"encrypted_value"`
	UpdatedBy      string    `db:"updated_by"`
	UpdateTime     time.Time `db:"update_time"`
}

type Model struct {
	conn sqlx.SqlConn
}

func NewModel(conn sqlx.SqlConn) *Model {
	return &Model{conn: conn}
}

func (m *Model) Load(ctx context.Context) (Record, error) {
	var record Record
	err := m.conn.QueryRowCtx(ctx, &record, `SELECT encrypted_value, updated_by, update_time
FROM telegram_plugin_configuration
WHERE id = 1`)
	return record, err
}

func (m *Model) Save(ctx context.Context, encryptedValue, updatedBy string) error {
	_, err := m.conn.ExecCtx(ctx, `INSERT INTO telegram_plugin_configuration (id, encrypted_value, updated_by)
VALUES (1, $1, $2)
ON CONFLICT (id) DO UPDATE SET
  encrypted_value = EXCLUDED.encrypted_value,
  updated_by = EXCLUDED.updated_by,
  update_time = CURRENT_TIMESTAMP`, encryptedValue, updatedBy)
	return err
}

func (m *Model) Delete(ctx context.Context) error {
	_, err := m.conn.ExecCtx(ctx, "DELETE FROM telegram_plugin_configuration WHERE id = 1")
	return err
}
