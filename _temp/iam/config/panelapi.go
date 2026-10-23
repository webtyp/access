package config

import (
	"webtyp.com/model"
)

type AdminMeResponse struct {
	Email       string
	Name        string
	IsAdmin     bool
	HasPasskey  bool
	MultiFactor bool
}

func (m *AdminMeResponse) IsNil() bool { return m == nil }

func (m *AdminMeResponse) EncodeFields(w model.FieldWriter) {
	w.String("email", m.Email)
	w.String("name", m.Name)
	w.Bool("is_admin", m.IsAdmin)
	w.Bool("has_passkey", m.HasPasskey)
	w.Bool("multi_factor", m.MultiFactor)
}

func (m *AdminMeResponse) DecodeFields(r model.FieldReader) {
	m.Email, _ = r.String("email")
	m.Name, _ = r.String("name")
	m.IsAdmin, _ = r.Bool("is_admin")
	m.HasPasskey, _ = r.Bool("has_passkey")
	m.MultiFactor, _ = r.Bool("multi_factor")
}

type AdminProjectView struct {
	Id        string
	Name      string
	Active    bool
	CreatedAt int64
}

func (m *AdminProjectView) IsNil() bool { return m == nil }

func (m *AdminProjectView) EncodeFields(w model.FieldWriter) {
	w.String("id", m.Id)
	w.String("name", m.Name)
	w.Bool("active", m.Active)
	w.Int("created_at", m.CreatedAt)
}

func (m *AdminProjectView) DecodeFields(r model.FieldReader) {
	m.Id, _ = r.String("id")
	m.Name, _ = r.String("name")
	m.Active, _ = r.Bool("active")
	m.CreatedAt, _ = r.Int("created_at")
}

type AdminProjectsResponse struct {
	Projects []AdminProjectView
}

func (m *AdminProjectsResponse) IsNil() bool { return m == nil }

func (m *AdminProjectsResponse) EncodeFields(w model.FieldWriter) {
	arr := w.Array("projects", len(m.Projects))
	for i := range m.Projects {
		arr.Object(&m.Projects[i])
	}
	arr.Close()
}

func (m *AdminProjectsResponse) DecodeFields(r model.FieldReader) {
	if arr, ok := r.Array("projects"); ok {
		m.Projects = make([]AdminProjectView, arr.Len())
		for i := 0; i < arr.Len(); i++ {
			arr.Object(i, &m.Projects[i])
		}
	}
}

type AdminCreateProjectRequest struct {
	Name      string
	PublicKey string
}

func (m *AdminCreateProjectRequest) IsNil() bool { return m == nil }

func (m *AdminCreateProjectRequest) Schema() []model.Field {
	return []model.Field{
		{Name: "name", Type: model.Text()},
		{Name: "public_key", Type: model.Text()},
	}
}

func (m *AdminCreateProjectRequest) Pointers() []any {
	return []any{&m.Name, &m.PublicKey}
}

func (m *AdminCreateProjectRequest) EncodeFields(w model.FieldWriter) {
	w.String("name", m.Name)
	w.String("public_key", m.PublicKey)
}

func (m *AdminCreateProjectRequest) DecodeFields(r model.FieldReader) {
	m.Name, _ = r.String("name")
	m.PublicKey, _ = r.String("public_key")
}

type AdminProjectKeyRequest struct {
	ProjectId string
	PublicKey string
}

func (m *AdminProjectKeyRequest) IsNil() bool { return m == nil }

func (m *AdminProjectKeyRequest) EncodeFields(w model.FieldWriter) {
	w.String("project_id", m.ProjectId)
	w.String("public_key", m.PublicKey)
}

func (m *AdminProjectKeyRequest) DecodeFields(r model.FieldReader) {
	m.ProjectId, _ = r.String("project_id")
	m.PublicKey, _ = r.String("public_key")
}

type AdminProjectIDRequest struct {
	ProjectId string
}

func (m *AdminProjectIDRequest) IsNil() bool { return m == nil }

func (m *AdminProjectIDRequest) EncodeFields(w model.FieldWriter) {
	w.String("project_id", m.ProjectId)
}

func (m *AdminProjectIDRequest) DecodeFields(r model.FieldReader) {
	m.ProjectId, _ = r.String("project_id")
}

type AdminSetActiveRequest struct {
	ProjectId string
	Active    bool
}

func (m *AdminSetActiveRequest) IsNil() bool { return m == nil }

func (m *AdminSetActiveRequest) EncodeFields(w model.FieldWriter) {
	w.String("project_id", m.ProjectId)
	w.Bool("active", m.Active)
}

func (m *AdminSetActiveRequest) DecodeFields(r model.FieldReader) {
	m.ProjectId, _ = r.String("project_id")
	m.Active, _ = r.Bool("active")
}

type RoleView struct {
	Code        string
	Name        string
	Description string
	SessionTtl  int64
	UserCount   int64
}

func (m *RoleView) IsNil() bool { return m == nil }

func (m *RoleView) EncodeFields(w model.FieldWriter) {
	w.String("code", m.Code)
	w.String("name", m.Name)
	w.String("description", m.Description)
	w.Int("session_ttl", m.SessionTtl)
	w.Int("user_count", m.UserCount)
}

func (m *RoleView) DecodeFields(r model.FieldReader) {
	m.Code, _ = r.String("code")
	m.Name, _ = r.String("name")
	m.Description, _ = r.String("description")
	m.SessionTtl, _ = r.Int("session_ttl")
	m.UserCount, _ = r.Int("user_count")
}

type AdminRolesResponse struct {
	Roles []RoleView
}

func (m *AdminRolesResponse) IsNil() bool { return m == nil }

func (m *AdminRolesResponse) EncodeFields(w model.FieldWriter) {
	arr := w.Array("roles", len(m.Roles))
	for i := range m.Roles {
		arr.Object(&m.Roles[i])
	}
	arr.Close()
}

func (m *AdminRolesResponse) DecodeFields(r model.FieldReader) {
	if arr, ok := r.Array("roles"); ok {
		m.Roles = make([]RoleView, arr.Len())
		for i := 0; i < arr.Len(); i++ {
			arr.Object(i, &m.Roles[i])
		}
	}
}

type AdminCreateRoleRequest struct {
	ProjectId   string
	Code        string
	Name        string
	Description string
}

func (m *AdminCreateRoleRequest) IsNil() bool { return m == nil }

func (m *AdminCreateRoleRequest) Schema() []model.Field {
	return []model.Field{
		{Name: "code", Type: model.Text()},
		{Name: "name", Type: model.Text()},
		{Name: "description", Type: model.Text()},
	}
}

func (m *AdminCreateRoleRequest) Pointers() []any {
	return []any{&m.Code, &m.Name, &m.Description}
}

func (m *AdminCreateRoleRequest) EncodeFields(w model.FieldWriter) {
	w.String("project_id", m.ProjectId)
	w.String("code", m.Code)
	w.String("name", m.Name)
	w.String("description", m.Description)
}

func (m *AdminCreateRoleRequest) DecodeFields(r model.FieldReader) {
	m.ProjectId, _ = r.String("project_id")
	m.Code, _ = r.String("code")
	m.Name, _ = r.String("name")
	m.Description, _ = r.String("description")
}

type AdminRoleTTLRequest struct {
	ProjectId  string
	Code       string
	SessionTtl int64
}

func (m *AdminRoleTTLRequest) IsNil() bool { return m == nil }

func (m *AdminRoleTTLRequest) EncodeFields(w model.FieldWriter) {
	w.String("project_id", m.ProjectId)
	w.String("code", m.Code)
	w.Int("session_ttl", m.SessionTtl)
}

func (m *AdminRoleTTLRequest) DecodeFields(r model.FieldReader) {
	m.ProjectId, _ = r.String("project_id")
	m.Code, _ = r.String("code")
	m.SessionTtl, _ = r.Int("session_ttl")
}

type AdminRoleRefRequest struct {
	ProjectId string
	Code      string
}

func (m *AdminRoleRefRequest) IsNil() bool { return m == nil }

func (m *AdminRoleRefRequest) EncodeFields(w model.FieldWriter) {
	w.String("project_id", m.ProjectId)
	w.String("code", m.Code)
}

func (m *AdminRoleRefRequest) DecodeFields(r model.FieldReader) {
	m.ProjectId, _ = r.String("project_id")
	m.Code, _ = r.String("code")
}

type RoleUser struct {
	Email string
	Name  string
	Sub   string
}

func (m *RoleUser) IsNil() bool { return m == nil }

func (m *RoleUser) EncodeFields(w model.FieldWriter) {
	w.String("email", m.Email)
	w.String("name", m.Name)
	w.String("sub", m.Sub)
}

func (m *RoleUser) DecodeFields(r model.FieldReader) {
	m.Email, _ = r.String("email")
	m.Name, _ = r.String("name")
	m.Sub, _ = r.String("sub")
}

type AdminRoleUsersResponse struct {
	Users []RoleUser
}

func (m *AdminRoleUsersResponse) IsNil() bool { return m == nil }

func (m *AdminRoleUsersResponse) EncodeFields(w model.FieldWriter) {
	arr := w.Array("users", len(m.Users))
	for i := range m.Users {
		arr.Object(&m.Users[i])
	}
	arr.Close()
}

func (m *AdminRoleUsersResponse) DecodeFields(r model.FieldReader) {
	if arr, ok := r.Array("users"); ok {
		m.Users = make([]RoleUser, arr.Len())
		for i := 0; i < arr.Len(); i++ {
			arr.Object(i, &m.Users[i])
		}
	}
}

type AdminUserRoleRequest struct {
	ProjectId string
	Code      string
	Email     string
}

func (m *AdminUserRoleRequest) IsNil() bool { return m == nil }

func (m *AdminUserRoleRequest) Schema() []model.Field {
	return []model.Field{
		{Name: "project_id", Type: model.Text()},
		{Name: "code", Type: model.Text()},
		{Name: "email", Type: model.Text()},
	}
}

func (m *AdminUserRoleRequest) Pointers() []any {
	return []any{&m.ProjectId, &m.Code, &m.Email}
}

func (m *AdminUserRoleRequest) EncodeFields(w model.FieldWriter) {
	w.String("project_id", m.ProjectId)
	w.String("code", m.Code)
	w.String("email", m.Email)
}

func (m *AdminUserRoleRequest) DecodeFields(r model.FieldReader) {
	m.ProjectId, _ = r.String("project_id")
	m.Code, _ = r.String("code")
	m.Email, _ = r.String("email")
}

type AdminAssignResponse struct {
	Sub string
}

func (m *AdminAssignResponse) IsNil() bool { return m == nil }

func (m *AdminAssignResponse) EncodeFields(w model.FieldWriter) {
	w.String("sub", m.Sub)
}

func (m *AdminAssignResponse) DecodeFields(r model.FieldReader) {
	m.Sub, _ = r.String("sub")
}

type AdminAuditEntry struct {
	ActorEmail string
	Action     string
	Target     string
	Detail     string
	CreatedAt  int64
}

func (m *AdminAuditEntry) IsNil() bool { return m == nil }

func (m *AdminAuditEntry) EncodeFields(w model.FieldWriter) {
	w.String("actor_email", m.ActorEmail)
	w.String("action", m.Action)
	w.String("target", m.Target)
	w.String("detail", m.Detail)
	w.Int("created_at", m.CreatedAt)
}

func (m *AdminAuditEntry) DecodeFields(r model.FieldReader) {
	m.ActorEmail, _ = r.String("actor_email")
	m.Action, _ = r.String("action")
	m.Target, _ = r.String("target")
	m.Detail, _ = r.String("detail")
	m.CreatedAt, _ = r.Int("created_at")
}

type AdminAuditResponse struct {
	Entries []AdminAuditEntry
}

func (m *AdminAuditResponse) IsNil() bool { return m == nil }

func (m *AdminAuditResponse) EncodeFields(w model.FieldWriter) {
	arr := w.Array("entries", len(m.Entries))
	for i := range m.Entries {
		arr.Object(&m.Entries[i])
	}
	arr.Close()
}

func (m *AdminAuditResponse) DecodeFields(r model.FieldReader) {
	if arr, ok := r.Array("entries"); ok {
		m.Entries = make([]AdminAuditEntry, arr.Len())
		for i := 0; i < arr.Len(); i++ {
			arr.Object(i, &m.Entries[i])
		}
	}
}
