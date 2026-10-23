//go:build wasm

package panel

import (
	"webtyp.com/auth/passkey"
	"webtyp.com/base64"
	"webtyp.com/components/actionbutton"
	"webtyp.com/dom"
	"webtyp.com/fetch"
	"webtyp.com/html"
	"webtyp.com/json"
	"webtyp.com/model"
	"webtyp.com/webauthn"
)

type passkeyRegisterOptions struct {
	Challenge string
	RPID      string
	UserID    string
	UserName  string
}

func (o *passkeyRegisterOptions) IsNil() bool { return o == nil }
func (o *passkeyRegisterOptions) DecodeFields(r model.FieldReader) {
	o.Challenge, _ = r.String("challenge")
	o.RPID, _ = r.String("rpId")
	o.UserID, _ = r.String("userId")
	o.UserName, _ = r.String("userName")
}

type passkeyVerifyOptions struct {
	Challenge        string
	RPID             string
	AllowCredentials []string
}

func (o *passkeyVerifyOptions) IsNil() bool { return o == nil }
func (o *passkeyVerifyOptions) DecodeFields(r model.FieldReader) {
	o.Challenge, _ = r.String("challenge")
	o.RPID, _ = r.String("rpId")
	if arr, ok := r.Array("allowCredentials"); ok {
		n := arr.Len()
		o.AllowCredentials = make([]string, n)
		for i := 0; i < n; i++ {
			o.AllowCredentials[i] = arr.String(i)
		}
	}
}

// ShowEnrollment renders the passkey enrollment view.
func ShowEnrollment(reload func()) {
	btn := &actionbutton.ActionButton{
		Text:    "Registrar passkey",
		Variant: actionbutton.Primary,
		OnClick: func(e dom.Event) {
			go Enroll(reload)
		},
	}
	box := html.Div().ID("iam-auth-gate").Class("iam-auth-gate").Child(
		html.H2().Text("Registro de passkey"),
		html.P().Text("Tu cuenta requiere registrar una passkey en este dispositivo para administrar iam."),
		btn,
	)
	_ = dom.Render("app", box)
}

// ShowStepUp renders the MFA step-up verification view.
func ShowStepUp(reload func()) {
	btn := &actionbutton.ActionButton{
		Text:    "Verificar con passkey",
		Variant: actionbutton.Primary,
		OnClick: func(e dom.Event) {
			go Verify(reload)
		},
	}
	box := html.Div().ID("iam-auth-gate").Class("iam-auth-gate").Child(
		html.H2().Text("Verificación requerida"),
		html.P().Text("Se requiere verificación de segundo factor con tu passkey para continuar."),
		btn,
	)
	_ = dom.Render("app", box)
}

func Enroll(reload func()) {
	type fetchRes struct {
		resp *fetch.Response
		err  error
	}
	ch := make(chan fetchRes, 1)
	fetch.Post(passkey.PathRegisterOptions).Send(func(resp *fetch.Response, err error) {
		ch <- fetchRes{resp, err}
	})
	res := <-ch
	if res.err != nil || res.resp == nil {
		ShowStatus("No se pudo contactar al servidor.")
		return
	}
	if res.resp.Status == 401 {
		authHdr := res.resp.GetHeader("WWW-Authenticate")
		if authHdr == "" {
			authHdr = res.resp.GetHeader("www-authenticate")
		}
		if authHdr == passkey.ChallengeRecentLogin {
			ShowStatus("Inicia sesión de nuevo para registrar tu passkey.")
			if ref, ok := dom.Get("login"); ok {
				ref.SetAttr("style", "")
			}
			return
		}
		ShowStatus("No autorizado para registrar passkey.")
		return
	}
	if res.resp.Status != 200 {
		ShowStatus("Error al obtener opciones de registro de passkey.")
		return
	}

	var opts passkeyRegisterOptions
	if err := json.Decode(res.resp.Body(), &opts); err != nil {
		ShowStatus("Respuesta ilegible del servidor.")
		return
	}

	chalBytes, _ := base64.URLDecode(opts.Challenge)
	uidBytes, _ := base64.URLDecode(opts.UserID)

	cred, err := webauthn.Create(webauthn.CreateOptions{
		RPID:             opts.RPID,
		RPName:           "iam",
		UserID:           uidBytes,
		UserName:         opts.UserName,
		UserDisplayName:  opts.UserName,
		Challenge:        chalBytes,
		ResidentKey:      true,
		UserVerification: "required",
	})
	if err != nil {
		ShowStatus("El registro de passkey no se completó.")
		return
	}

	chReg := make(chan fetchRes, 1)
	fetch.Post(passkey.PathRegister).Body([]byte(cred.RawResponse)).Send(func(resp *fetch.Response, err error) {
		chReg <- fetchRes{resp, err}
	})
	resReg := <-chReg
	if resReg.err != nil || resReg.resp == nil || resReg.resp.Status != 204 {
		ShowStatus("Error al registrar passkey.")
		return
	}

	// Then run verify
	Verify(reload)
}

func Verify(reload func()) {
	type fetchRes struct {
		resp *fetch.Response
		err  error
	}
	ch := make(chan fetchRes, 1)
	fetch.Post(passkey.PathVerifyOptions).Send(func(resp *fetch.Response, err error) {
		ch <- fetchRes{resp, err}
	})
	res := <-ch
	if res.err != nil || res.resp == nil {
		ShowStatus("No se pudo contactar al servidor.")
		return
	}
	if res.resp.Status != 200 {
		ShowStatus("Error al obtener opciones de verificación.")
		return
	}

	var opts passkeyVerifyOptions
	if err := json.Decode(res.resp.Body(), &opts); err != nil {
		ShowStatus("Respuesta ilegible del servidor.")
		return
	}

	chalBytes, _ := base64.URLDecode(opts.Challenge)
	var allowCreds [][]byte
	for _, idStr := range opts.AllowCredentials {
		b, _ := base64.URLDecode(idStr)
		allowCreds = append(allowCreds, b)
	}

	assertion, err := webauthn.Get(webauthn.GetOptions{
		RPID:             opts.RPID,
		Challenge:        chalBytes,
		AllowCredentials: allowCreds,
		UserVerification: "required",
	})
	if err != nil {
		ShowStatus("La verificación de passkey no se completó.")
		return
	}

	chVer := make(chan fetchRes, 1)
	fetch.Post(passkey.PathVerify).Body([]byte(assertion.RawResponse)).Send(func(resp *fetch.Response, err error) {
		chVer <- fetchRes{resp, err}
	})
	resVer := <-chVer
	if resVer.err != nil || resVer.resp == nil || resVer.resp.Status != 204 {
		ShowStatus("Error al verificar passkey.")
		return
	}

	HideStatus()
	if ref, ok := dom.Get("iam-auth-gate"); ok {
		ref.SetAttr("style", "display: none;")
	}
	if reload != nil {
		reload()
	}
}
