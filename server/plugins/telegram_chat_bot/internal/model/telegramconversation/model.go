package telegramconversation

import (
	"context"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type Model struct {
	conn sqlx.SqlConn
}

func NewModel(conn sqlx.SqlConn) *Model {
	return &Model{conn: conn}
}

func (m *Model) FindConversation(ctx context.Context, chatID int64) (string, error) {
	var conversationID string
	err := m.conn.QueryRowCtx(ctx, &conversationID,
		"SELECT conversation_id FROM telegram_chat_conversation WHERE chat_id = $1", chatID)
	if errors.Is(err, sqlx.ErrNotFound) {
		return "", nil
	}
	return conversationID, err
}

func (m *Model) SaveConversation(ctx context.Context, chatID int64, conversationID, title string) error {
	_, err := m.conn.ExecCtx(ctx, `INSERT INTO telegram_chat_conversation (chat_id, conversation_id, title)
VALUES ($1, $2, $3)
ON CONFLICT (chat_id) DO UPDATE SET
  conversation_id = EXCLUDED.conversation_id,
  title = EXCLUDED.title,
  update_time = CURRENT_TIMESTAMP`, chatID, conversationID, title)
	return err
}

func (m *Model) CountConversations(ctx context.Context) (int64, error) {
	var count int64
	err := m.conn.QueryRowCtx(ctx, &count, "SELECT COUNT(*) FROM telegram_chat_conversation")
	return count, err
}

func (m *Model) LastProcessedUpdateID(ctx context.Context) (int64, error) {
	var updateID int64
	err := m.conn.QueryRowCtx(ctx, &updateID,
		"SELECT COALESCE(MAX(update_id), 0) FROM telegram_processed_update")
	return updateID, err
}

func (m *Model) MarkUpdateProcessed(ctx context.Context, updateID int64) (bool, error) {
	result, err := m.conn.ExecCtx(ctx,
		"INSERT INTO telegram_processed_update (update_id) VALUES ($1) ON CONFLICT (update_id) DO NOTHING",
		updateID,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}

func (m *Model) PruneProcessedUpdates(ctx context.Context, before time.Time) error {
	_, err := m.conn.ExecCtx(ctx,
		"DELETE FROM telegram_processed_update WHERE processed_at < $1", before)
	return err
}
