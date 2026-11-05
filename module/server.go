//go:build !browser

package module

import (
	"errors"

	"webtyp.com/access"
	"webtyp.com/appkit"
	"webtyp.com/auth"
	"webtyp.com/auth/authority"
	trustedip "webtyp.com/auth/trusted_ip"
	"webtyp.com/env"
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/rbac"
	"webtyp.com/router"
	"webtyp.com/time"
)

const (
	EnvSessionIdleMinutes = "SESSION_IDLE_MINUTES"
	maxLoginAttempts      = 3
	loginWindow           = 15 * 60 // seconds
	loginBlock            = 15 * 60
	RoleAdmin             = model.RoleCode("admin")
	permWildcardAll       = "wildcard:all"
	defaultIdleMinutes    = 30
)

type eventBus interface {
	Subscribe(topic string, h events.Handler) events.Subscription
}

func securityEventLogger(logger func(...any)) func(auth.SecurityEvent) {
	return func(e auth.SecurityEvent) {
		if logger == nil || e.Type == auth.EventDevAutologin {
			return
		}
		args := []any{"app: SECURITY EVENT — type:", e.Type}
		for _, f := range []struct{ label, value string }{
			{"userID:", e.UserID},
			{"resource:", e.Resource},
			{"IP:", e.IP},
		} {
			if f.value != "" {
				args = append(args, f.label, f.value)
			}
		}
		logger(args...)
	}
}

func seedFirstAdmin(d appkit.Deps, authMod *authority.Module, rbacSvc *rbac.Service, accessMod *access.Module, code, name, ip string) error {
	user, err := authMod.CreateUser("", name, "")
	if err != nil {
		return err
	}
	if err := authMod.RegisterLAN(user.Id, code); err != nil {
		return err
	}
	switch reg, err := appkit.Find[FirstAdminRegistrar](d); {
	case err == nil:
		if err := reg.RegisterFirstAdmin(user.Id, code, name, ip); err != nil {
			return err
		}
	case !errors.Is(err, appkit.ErrNoProvider):
		return err
	}
	if err := accessMod.EnsureRoleFor(RoleAdmin, "Admin", "System administrator", []string{user.Id}); err != nil {
		return err
	}
	role, err := rbacSvc.GetRoleByCode(d.TenantID, RoleAdmin)
	if err != nil {
		return err
	}
	if err := rbacSvc.CreatePermission(d.TenantID, permWildcardAll, permWildcardAll, model.Wildcard, model.AllActions); err != nil {
		return err
	}
	return rbacSvc.AssignPermission(d.TenantID, role.Id, permWildcardAll)
}

// identity is the people modules: roles, users and their administration.
type identity struct {
	roles  *rbac.Service
	users  *authority.Module
	access *access.Module
}

func buildIdentity(d appkit.Deps, logger func(...any)) (identity, error) {
	rbacSvc, err := rbac.New(d.DB)
	if err != nil {
		return identity{}, err
	}

	idleMinutes := defaultIdleMinutes
	if v := env.Get(EnvSessionIdleMinutes); v != "" {
		if n, err := fmt.Convert(v).Int(); err == nil && n > 0 {
			idleMinutes = n
		} else {
			if logger != nil {
				logger("access: invalid "+EnvSessionIdleMinutes+", using", defaultIdleMinutes, "minutes")
			}
		}
	}

	authMod, err := authority.New(d.DB, auth.Config{
		IDs:         d.IDs,
		Events:      d.Events,
		IdleTTL:     time.Unix(int64(idleMinutes)) * time.Minute,
		Permissions: &auth.Permissions{Grants: rbacSvc, ProjectID: d.TenantID},
	})
	if err != nil {
		return identity{}, err
	}
	authMod.SetLog(logger)

	accessMod, err := access.New(access.Deps{
		DB: d.DB, IDs: d.IDs, Users: authMod, Roles: rbacSvc, ProjectID: d.TenantID,
		Assurance: model.AssuranceSingleFactor,
	})
	if err != nil {
		return identity{}, err
	}

	return identity{roles: rbacSvc, users: authMod, access: accessMod}, nil
}

type entry struct {
	users  *authority.Module
	access *access.Module
	authn  router.Middleware
	authz  model.Authorizer
	policy model.PolicyDescriber
}

// lazyTrustedStore asks for the staff's devices at sign-in time: the staff module is built after this one.
type lazyTrustedStore struct {
	d appkit.Deps
}

func (l lazyTrustedStore) IsTrustedIP(userID, ip string) bool {
	s, err := appkit.Find[StaffDevices](l.d)
	if err != nil {
		return false
	}
	return s.IsTrustedIP(userID, ip)
}

func (e *entry) ModelName() string                  { return "access_entry" }
func (e *entry) Authn() router.Middleware            { return e.authn }
func (e *entry) Authorize() model.Authorizer         { return e.authz }
func (e *entry) Policy() model.PolicyDescriber       { return e.policy }
func (e *entry) Parts() []model.ModuleNaming         { return []model.ModuleNaming{e.users, e.access} }

var server = appkit.ModuleServer{Build: func(d appkit.Deps) (model.ModuleNaming, error) { return build(d) }}

var browser appkit.ModuleBrowser

func build(d appkit.Deps) (model.ModuleNaming, error) {
	logger := func(...any) {}
	if bus, ok := d.Events.(eventBus); ok {
		logSecurityEvent := securityEventLogger(logger)
		bus.Subscribe(auth.TopicSecurity, func(e events.Event) {
			if se, ok := e.Payload.(*auth.SecurityEvent); ok {
				logSecurityEvent(*se)
			}
		})
	}

	id, err := buildIdentity(d, logger)
	if err != nil {
		return nil, err
	}

	trustedStore := lazyTrustedStore{d: d}

	loginLimiter := auth.NewIPLimiter(maxLoginAttempts, loginWindow, loginBlock)
	id.users.Enable(trustedip.New(id.users, trustedStore, id.users, id.users, false,
		trustedip.WithRateLimit(loginLimiter),
		trustedip.WithFirstAdminSetup(id.users, func(code, name, ip string) error {
			return seedFirstAdmin(d, id.users, id.roles, id.access, code, name, ip)
		})))

	switch hasAdmin, err := id.users.HasIdentity(auth.ProviderTrustedIP); {
	case err != nil:
		logger("access: the database does not answer the administrators query — run go run ./migrate once. Detail:", err)
	case !hasAdmin:
		logger("access: no administrator yet — open the app in the browser to complete the first setup")
	}

	return &entry{
		users:  id.users,
		access: id.access,
		authn:  id.users.Authenticate(),
		authz: func(userID string, resource model.Resource, action model.Action) bool {
			return id.roles.Can(d.TenantID, userID, resource, action)
		},
		policy: nil,
	}, nil
}
