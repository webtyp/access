//go:build wasm

package panel

import (
	"webtyp.com/dom"
	"webtyp.com/fetch"
	"webtyp.com/html"
	"webtyp.com/router"
)

var onMFARequired func()

// SetOnMFARequired registers a handler to call when any panel request gets a 401 ChallengeMFARequired.
func SetOnMFARequired(fn func()) {
	onMFARequired = fn
}

// Fetch sends a request and intercepts 401 ChallengeMFARequired to switch to step-up view.
func Fetch(req *fetch.Request, callback func(*fetch.Response, error)) {
	req.Send(func(resp *fetch.Response, err error) {
		if err == nil && resp != nil && resp.Status == 401 {
			authHdr := resp.GetHeader("WWW-Authenticate")
			if authHdr == "" {
				authHdr = resp.GetHeader("www-authenticate")
			}
			if authHdr == router.ChallengeMFARequired {
				if onMFARequired != nil {
					onMFARequired()
					return
				}
			}
		}
		if callback != nil {
			callback(resp, err)
		}
	})
}

func ShowStatus(msg string) {
	ref, ok := dom.Get("iam-status-message")
	if !ok {
		el := html.Div().ID("iam-status-message").Class("iam-status-banner").Text(msg)
		_ = dom.Append("body", el)
	} else {
		ref.SetText(msg)
	}
}

func HideStatus() {
	ref, ok := dom.Get("iam-status-message")
	if ok {
		ref.SetText("")
	}
}
