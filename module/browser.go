//go:build browser

package module

import (
	accessui "webtyp.com/access/ui"
	"webtyp.com/appkit"
	"webtyp.com/auth"
	. "webtyp.com/dom"
	"webtyp.com/fetch"
	"webtyp.com/fmt"
	"webtyp.com/form"
	"webtyp.com/json"
	"webtyp.com/lang"
	"webtyp.com/layout/login"
	"webtyp.com/model"
	"webtyp.com/unixid"
	"webtyp.com/view"
)

var server appkit.ModuleServer

const defaultIdleMinutes = 30

const (
	textKey         lang.Text = "Key"
	textSignIn      lang.Text = "Sign in"
	textName        lang.Text = "Name"
	textCreateAdmin lang.Text = "Create administrator"
	textSetupFailed lang.Text = "The administrator could not be created. Check the RUT entered."
	textLoginDenied lang.Text = "Access not authorized on this device"
	labelRUT                  = "RUT"
)

const LoginWrapperID = "login-wrapper"

func LoginForm(data *auth.RUTLoginData, ids model.IDGenerator) (*form.Form, error) {
	f, err := form.New(LoginWrapperID, data, ids)
	if err != nil {
		return nil, err
	}
	if inp := f.Input("code"); inp != nil {
		if setter, ok := inp.(interface{ SetSkipValidation(bool) }); ok {
			setter.SetSkipValidation(true)
		}
		if masker, ok := inp.(interface{ SetMasked(bool) }); ok {
			masker.SetMasked(true)
		}
		if titler, ok := inp.(interface{ SetTitle(string) }); ok {
			titler.SetTitle(lang.Translate(textKey).String())
		}
	}
	f.SubmitLabel(lang.Translate(textSignIn).String())
	return f, nil
}

func LoginScreen(b appkit.Brand, f *form.Form) *login.Login {
	return &login.Login{
		Title:    lang.Text(b.Name),
		Subtitle: "Enter your key to continue",
		Form:     f.Render(),
		LogoMark: b.LogoMark,
	}
}

func post(path string, data model.Encodable, done func(error), refused func()) {
	var out []byte
	if err := json.Encode(data, &out); err != nil {
		done(err)
		return
	}
	fetch.Post(path).ContentTypeJSON().Body(out).Send(func(resp *fetch.Response, err error) {
		if err != nil {
			done(err)
			return
		}
		if resp.Status != 200 {
			refused()
			return
		}
		Navigate("")
		done(nil)
	})
}

var browser = appkit.ModuleBrowser{
	Screens: func(d appkit.BrowserDeps) []view.Screen {
		return accessui.Screens(d.Caller)
	},
	Entry: func(d appkit.BrowserDeps) appkit.EntryScreens {
		ids, err := unixid.NewUnixID()
		if err != nil {
			panic(err)
		}

		loginMessage := NewString("")
		setupMessage := NewString("")

		rutData := &auth.RUTLoginData{}
		loginF, err := LoginForm(rutData, ids)
		if err == nil {
			loginF.OnSubmit(func(fd model.Fielder, done func(error)) {
				post(auth.PathLoginRUT, rutData, done, func() {
					msg := lang.Translate(textLoginDenied).String()
					loginMessage.Set(msg)
					done(fmt.Err(msg))
				})
			})
		}
		loginScr := LoginScreen(d.Brand, loginF)
		loginScr.Message = loginMessage

		setupData := &auth.SetupData{}
		setupForm, errSetup := form.New("setup-wrapper", setupData, ids)
		if errSetup == nil {
			if inp := setupForm.Input("code"); inp != nil {
				if titler, ok := inp.(interface{ SetTitle(string) }); ok {
					titler.SetTitle(labelRUT)
				}
				if setter, ok := inp.(interface{ SetSkipValidation(bool) }); ok {
					setter.SetSkipValidation(true)
				}
			}
			if inp := setupForm.Input("name"); inp != nil {
				if titler, ok := inp.(interface{ SetTitle(string) }); ok {
					titler.SetTitle(lang.Translate(textName).String())
				}
			}
			setupForm.SubmitLabel(lang.Translate(textCreateAdmin).String())
			setupForm.OnSubmit(func(fd model.Fielder, done func(error)) {
				post(auth.PathSetup, setupData, done, func() {
					msg := lang.Translate(textSetupFailed).String()
					setupMessage.Set(msg)
					done(fmt.Err(msg))
				})
			})
		}
		setupScr := &login.Login{
			Title:    lang.Text(d.Brand.Name),
			Subtitle: "First setup: create the administrator",
			Form:     setupForm.Render(),
			LogoMark: d.Brand.LogoMark,
			Message:  setupMessage,
		}

		return appkit.EntryScreens{
			Login:       loginScr.Render(),
			Setup:       setupScr.Render(),
			IdleSeconds: defaultIdleMinutes * 60,
		}
	},
}
