package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/text/language"
)

// I18nConf is the configuration structure for i18n
type I18nConf struct {
	Dir string `json:",optional"`
}

// Translator is a struct storing translating data.
type Translator struct {
	bundle       *i18n.Bundle
	localizer    map[language.Tag]*i18n.Localizer
	supportLangs []language.Tag
}

// Trans used to translate any i18n string.
func (l *Translator) Trans(ctx context.Context, msgId string) string {
	message, err := l.MatchLocalizer(ctx.Value("lang").(string)).LocalizeMessage(&i18n.Message{ID: msgId})
	if err != nil {
		return msgId
	}

	if message == "" {
		return msgId
	}

	return message
}

func ParseTags(lang string) []language.Tag {
	tags, _, err := language.ParseAcceptLanguage(lang)
	if err != nil {
		logx.Errorw("parse accept-language failed", logx.Field("detail", err))
		return []language.Tag{language.Chinese}
	}

	return tags
}

// MatchLocalizer used to matcher the localizer in map
func (l *Translator) MatchLocalizer(lang string) *i18n.Localizer {
	tags := ParseTags(lang)
	for _, v := range tags {
		if val, ok := l.localizer[v]; ok {
			return val
		}
	}

	return l.localizer[language.Chinese]
}

// AddLanguageSupport adds supports for new language
func (l *Translator) AddLanguageSupport(lang language.Tag) {
	l.supportLangs = append(l.supportLangs, lang)
	l.localizer[lang] = i18n.NewLocalizer(l.bundle, lang.String())
}

// AddBundleFromFile adds new bundle into translator from file path.
func (l *Translator) AddBundleFromFile(path string) error {
	if _, err := l.bundle.LoadMessageFile(path); err != nil {
		return err
	}
	return nil
}

func NewTranslator(conf I18nConf) *Translator {
	trans := &Translator{localizer: make(map[language.Tag]*i18n.Localizer)}
	bundle := i18n.NewBundle(language.Chinese)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	trans.bundle = bundle
	trans.AddLanguageSupport(language.Chinese)

	if strings.TrimSpace(conf.Dir) == "" {
		return trans
	}

	var files []string
	if err := filepath.WalkDir(conf.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(path), ".json") {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		logx.Must(fmt.Errorf("failed to read i18n directory %s: %w", conf.Dir, err))
	}

	for _, path := range files {
		languageName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		trans.AddLanguageSupport(ParseTags(languageName)[0])
		if err := trans.AddBundleFromFile(path); err != nil {
			logx.Must(fmt.Errorf("failed to load i18n file %s: %w", path, err))
		}
	}

	return trans
}
