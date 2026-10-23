//go:build wasm

package panel

import (
	"webtyp.com/dom"
	"webtyp.com/fetch"
	"webtyp.com/fmt"
	"webtyp.com/form"
	"webtyp.com/html"
	"webtyp.com/json"
	"webtyp.com/model"
	"github.com/veltylabs/iam/config"
)

func wireUsers() {
	idGen, _ := config.NewIDs()
	f, err := form.New(IDUsersForm, &config.AdminUserRoleRequest{}, idGen)
	if err == nil && f != nil {
		f.SubmitLabel("Asignar rol")
		f.OnSubmit(func(data model.Fielder, done func(error)) {
			req, ok := data.(*config.AdminUserRoleRequest)
			if !ok || fmt.TrimSpace(req.ProjectId) == "" || fmt.TrimSpace(req.Code) == "" || fmt.TrimSpace(req.Email) == "" {
				done(fmt.Err("All fields (Project ID, Role Code, Email) are required"))
				return
			}
			var buf []byte
			if err := json.Encode(req, &buf); err != nil {
				done(err)
				return
			}
			Fetch(fetch.Post(config.PathAdminUserAssign).ContentTypeJSON().Body(buf), func(resp *fetch.Response, err error) {
				if err != nil || resp.Status != 200 {
					done(fmt.Err("Error assigning role"))
					return
				}
				var assignResp config.AdminAssignResponse
				if err := json.Decode(resp.Body(), &assignResp); err == nil {
					ShowStatus("Rol asignado. Sub: " + assignResp.Sub)
				}
				done(nil)
				loadRoleUsersList(req.ProjectId, req.Code)
			})
		})
		_ = dom.Render(IDUsersForm, f)
	}

	setContainerText(IDUsersList, "Ingresá un Project ID y Código de rol en el formulario para consultar sus usuarios o asignar.")
}

func loadRoleUsersList(projectID, code string) {
	if projectID == "" || code == "" {
		return
	}
	url := config.PathAdminRoleUsers + "?project_id=" + projectID + "&code=" + code
	Fetch(fetch.Get(url), func(resp *fetch.Response, err error) {
		if err != nil || resp.Status != 200 {
			setContainerText(IDUsersList, "Error al cargar la lista de usuarios del rol.")
			return
		}
		var usersResp config.AdminRoleUsersResponse
		if err := json.Decode(resp.Body(), &usersResp); err != nil {
			setContainerText(IDUsersList, "Error decodificando usuarios.")
			return
		}

		table := html.Table().Class("iam-table").Child(
			html.Thead().Child(
				html.Tr().Child(
					html.Th().Text("Email"),
					html.Th().Text("Nombre"),
					html.Th().Text("Sub"),
					html.Th().Text("Acciones"),
				),
			),
		)
		tbody := html.Tbody()
		for _, u := range usersResp.Users {
			uEmail := u.Email
			btnRevoke := html.Button().Text("Revocar").Class("iam-btn-danger")
			btnRevoke.OnClick(func(event dom.Event) {
				req := &config.AdminUserRoleRequest{ProjectId: projectID, Code: code, Email: uEmail}
				var buf []byte
				_ = json.Encode(req, &buf)
				Fetch(fetch.Post(config.PathAdminUserRevoke).ContentTypeJSON().Body(buf), func(res *fetch.Response, err error) {
					if err == nil && res.Status == 200 {
						loadRoleUsersList(projectID, code)
					}
				})
			})

			row := html.Tr().Child(
				html.Td().Text(u.Email),
				html.Td().Text(u.Name),
				html.Td().Text(u.Sub),
				html.Td().Child(btnRevoke),
			)
			tbody.Child(row)
		}
		table.Child(tbody)
		_ = dom.Render(IDUsersList, table)
	})
}
