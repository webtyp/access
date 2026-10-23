package config

import (
	"webtyp.com/auth/authority"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/router"
)

const (
	HeaderOrigin       = "Origin"
	HeaderSecFetchSite = "Sec-Fetch-Site"

	// secFetchSameOrigin es el único valor aceptable: "same-site" NO alcanza,
	// y ésa es exactamente la razón por la que este archivo existe — la
	// cookie SSO es Domain=.velty.cl, así que misitio.velty.cl es same-site
	// respecto de iam.velty.cl y podría disparar estas rutas con la sesión
	// del administrador adjunta.
	secFetchSameOrigin = "same-origin"
)

// auditDetailMax acota lo que se guarda de la cabecera Origin recibida: es
// entrada del atacante y no puede inflar una fila de la base sin techo.
const auditDetailMax = 200

// RequireSameOrigin envuelve un handler de mutación: 403 si la petición no
// vino del propio origen de iam.
func RequireSameOrigin(db *orm.DB, ids model.IDGenerator, authMod *authority.Module, expectedOrigin string, h router.HandlerFunc) router.HandlerFunc {
	return func(ctx router.Context) {
		fetchSite := ctx.GetHeader(HeaderSecFetchSite)
		origin := ctx.GetHeader(HeaderOrigin)
		if fetchSite == secFetchSameOrigin || (origin != "" && origin == expectedOrigin) {
			h(ctx)
			return
		}
		adminEmail := ""
		if uid := ctx.UserID(); uid != "" {
			if u, err := authMod.GetUser(uid); err == nil {
				adminEmail = u.Email
			}
		}
		detail := origin
		if len(detail) > auditDetailMax {
			detail = detail[:auditDetailMax]
		}
		if err := RecordAudit(db, ids, adminEmail, AuditOriginDenied, adminEmail, detail); err != nil {
			fmt.Println("audit:", err)
		}
		ctx.WriteStatus(403)
	}
}
