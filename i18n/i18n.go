package i18n

import (
	"strings"

	"github.com/Unknwon/i18n"
	"github.com/kataras/iris"
)

type i18nMiddleware struct {
	config Config
}

// Serve serves the request, the actual middleware's job is here
func (i *i18nMiddleware) Serve(ctx *iris.Context) {
	wasByCookie := false
	// try to get by url parameter
	language := ctx.URLParam(i.config.URLParameter)

	if language == "" {
		// then try to take the lang field from the cookie
		language = ctx.GetCookie("lang")

		if len(language) > 0 {
			wasByCookie = true
		} else {
			// try to get by the request headers(?)
			if langHeader := ctx.RequestHeader("Accept-Language"); i18n.IsExist(langHeader) {
				language = langHeader
			}
		}
	}
	// if it was not taken by the cookie, then set the cookie in order to have it
	if !wasByCookie {
		ctx.SetCookieKV("lang", language)
	}
	if language == "" {
		language = i.config.Default
	}
	locale := i18n.Locale{Lang: language}
	ctx.Set("language", language)
	ctx.Set("translate", locale.Tr)
	ctx.Next()
}

// New returns a new i18n middleware
func New(c Config) iris.HandlerFunc {
	if len(c.Languages) == 0 {
		panic("You cannot use this middleware without set the Languages option, please try again and read the docs.")
	}
	i := &i18nMiddleware{config: c}
	firstlanguage := ""
	//load the files
	for k, v := range c.Languages {
		if !strings.HasSuffix(v, ".ini") {
			v += ".ini"
		}
		err := i18n.SetMessage(k, v)
		if err != nil && err != i18n.ErrLangAlreadyExist {
			panic("Iris i18n Middleware: Failed to set locale file" + k + " Error:" + err.Error())
		}
		if firstlanguage == "" {
			firstlanguage = k
		}
	}
	// if not default language setted then set to the first of the i.options.Languages
	if c.Default == "" {
		c.Default = firstlanguage
	}

	i18n.SetDefaultLang(i.config.Default)
	return i.Serve
}
