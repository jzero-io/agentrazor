package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	coreconfig "github.com/jzero-io/agentrazor/core-engine/config"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	coreconfig.Config

	Rest RestConf
	Log  LogConf

	Banner       BannerConf
	Telegram     TelegramConf
	Conversation ConversationConf
}

type RestConf struct {
	rest.RestConf
}

type LogConf struct {
	logx.LogConf
}

type BannerConf struct {
	Text     string `json:",default=JZERO"`
	Color    string `json:",default=green"`
	FontName string `json:",default=starwars,options=big|larry3d|starwars|standard"`
}

type TelegramConf struct {
	BotToken              string `json:",optional"`
	GroupChatIDs          string `json:",optional"`
	APIBaseURL            string `json:",default=https://api.telegram.org"`
	PollTimeoutSeconds    int    `json:",default=25"`
	RequestTimeoutSeconds int    `json:",default=40"`
}

type ConversationConf struct {
	BaseURL        string `json:",default=http://127.0.0.1:8001"`
	APIKey         string `json:",optional"`
	TimeoutSeconds int    `json:",default=600"`
}

func (c TelegramConf) Enabled() bool {
	return strings.TrimSpace(c.BotToken) != ""
}

func (c TelegramConf) Validate() error {
	if !c.Enabled() {
		return nil
	}
	if _, err := c.AllowedGroupIDs(); err != nil {
		return err
	}
	if strings.TrimSpace(c.APIBaseURL) == "" {
		return errors.New("Telegram API base URL is required")
	}
	if c.PollTimeoutSeconds <= 0 || c.PollTimeoutSeconds > 50 {
		return errors.New("Telegram poll timeout must be between 1 and 50 seconds")
	}
	if c.RequestTimeoutSeconds <= 0 {
		return errors.New("Telegram request timeout must be positive")
	}
	return nil
}

func (c TelegramConf) AllowedGroupIDs() (map[int64]struct{}, error) {
	result := make(map[int64]struct{})
	for _, value := range strings.Split(c.GroupChatIDs, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id == 0 {
			return nil, fmt.Errorf("invalid Telegram group chat ID %q", value)
		}
		result[id] = struct{}{}
	}
	if c.Enabled() && len(result) == 0 {
		return nil, errors.New("at least one Telegram group chat ID is required")
	}
	return result, nil
}

func (c ConversationConf) Validate() error {
	if c.TimeoutSeconds <= 0 {
		return errors.New("AgentRazor request timeout must be positive")
	}
	return nil
}
