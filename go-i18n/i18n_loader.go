package i18n

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/kataras/iris/v12/i18n"

	"github.com/BurntSushi/toml"
	goI18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// NewLoader returns a Loader that loads the translate files using the go-i18n library which accepts yaml
// or toml format as its special templates. Read more at: https://github.com/nicksnyder/go-i18n.
//
// The pattern must follow the specs of https://golang.org/pkg/path/filepath/#Glob.
// The filenames matter, and should be formated as: "$/tag/$file" or "-$tag-" or "_$tag_" or "$name.$tag.$ext"
// in order to match it with the first argument's registered languages.
//
// app.I18n.Reset(NewLoader("./locales/*/*"), "en-US", "el-GR", "zh-CN")
func NewLoader(globPatternOrFilenames ...string) i18n.Loader {
	n := len(globPatternOrFilenames)
	if n == 0 {
		return nil
	}

	return func(m *i18n.Matcher) (i18n.Localizer, error) {
		fileNames := globPatternOrFilenames
		if globPattern := globPatternOrFilenames[0]; n == 1 {
			names, err := filepath.Glob(globPattern)
			if err != nil {
				return nil, err
			}

			fileNames = names
		}

		languageFiles, err := m.ParseLanguageFiles(fileNames)
		if err != nil {
			return nil, err
		}

		b := goI18n.NewBundle(m.Languages[0])
		locales := make(i18n.MemoryLocalizer)

		unmarshalFuncs := make(map[string]goI18n.UnmarshalFunc)

		for langIndex, langFiles := range languageFiles {
			tag := m.Languages[langIndex]
			for _, fileName := range langFiles {
				// take format, without dot.
				format := "yaml"
				if idx := strings.LastIndexByte(fileName, '.'); idx > 0 && len(fileName)-1 > idx+1 {
					format = fileName[idx+1:]
				}

				unmarshalFunc := yaml.Unmarshal
				switch format {
				case "toml", "tml":
					unmarshalFunc = toml.Unmarshal
				case "json":
					unmarshalFunc = json.Unmarshal
				}

				unmarshalFuncs[format] = unmarshalFunc

				buf, err := ioutil.ReadFile(fileName)
				if err != nil {
					return nil, err
				}

				messageFile, err := goI18n.ParseMessageFileBytes(buf, fileName, unmarshalFuncs)
				if err != nil {
					return nil, err
				}

				messageFile.Tag = tag

				if err := b.AddMessages(messageFile.Tag, messageFile.Messages...); err != nil {
					return nil, err
				}
			}

			// we register a new localizer for each known language,
			// i18n package matches the language by itself so we don't really need to do it at serve-time,
			// this is why we preload them here.
			locales[langIndex] = &goi18nLocale{
				index:     langIndex,
				id:        tag.String(),
				tag:       &tag,
				localizer: goI18n.NewLocalizer(b, tag.String()),
			}
		}

		if len(locales) == 0 {
			return nil, fmt.Errorf("locales not found in %s", strings.Join(globPatternOrFilenames, ", "))
		}

		return locales, nil
	}

}

type goi18nLocale struct {
	index     int
	id        string
	tag       *language.Tag
	localizer *goI18n.Localizer
}

func (l *goi18nLocale) Index() int {
	return l.index
}

func (l *goi18nLocale) Tag() *language.Tag {
	return l.tag
}

func (l *goi18nLocale) Language() string {
	return l.id
}

func (l *goi18nLocale) GetMessage(key string, args ...interface{}) string {
	message := &goI18n.Message{
		ID: key,
	}

	config := &goI18n.LocalizeConfig{}

	for _, arg := range args {
		switch v := arg.(type) {
		// case string:
		// 	if message.Description == "" {
		// 		message.Description = v
		// 	} else {
		// 		config.TemplateData = v
		// 	}
		case *goI18n.LocalizeConfig:
			config = v
		case template.FuncMap:
			config.Funcs = v
		default:
			if config.TemplateData != nil {
				config.PluralCount = v
			} else {
				config.TemplateData = v
			}
		}
	}

	config.DefaultMessage = message

	text, err := l.localizer.Localize(config)
	if err != nil {
		return err.Error()
	}

	return text
}
