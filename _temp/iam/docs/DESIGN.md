# Diseño — decisiones y alternativas descartadas

Este documento justifica **por qué** la arquitectura es la que es. La estructura
y el mecanismo están en [`ARCHITECTURE.md`](ARCHITECTURE.md) §6.4; no se
repiten aquí. Lo que sigue es el razonamiento — y, sobre todo, la alternativa
que se evaluó y se rechazó — para que nadie la reintente sin saber por qué.

---

## 1. `client_secret`, no un token — por qué el nombre importa

**Alternativa descartada:** llamar a la credencial estática del proyecto
consumidor `IAM_CLIENT_TOKEN` (o cualquier variante con "token") en vez de
`IAM_CLIENT_SECRET`.

### El motivo

Este mismo repo ya tiene **dos conceptos distintos** que un lector podría
confundir si compartieran nombre:

| | `client_secret` | "token" (lo que emite `iam`) |
|---|---|---|
| Qué prueba | **quién es la app** que llama (`misitio`, y no un impostor) | **quién es el usuario**, y qué puede hacer, en esta sesión |
| Vida | estática — se emite **una vez**, al registrar el proyecto | corta (TTL) — se reemite en cada login/refresco |
| Dónde vive | variable de entorno del **servidor** del proyecto consumidor, nunca en el navegador | cookie SSO / respuesta de `FetchAuthzToken`, viaja por request |
| Quién lo rota | el dueño del proyecto, a mano, cuando hace falta | `iam` mismo, automáticamente, todo el tiempo |

Si la credencial estática se llamara también "token", el código y la
documentación tendrían la misma palabra para dos cosas con vidas, dueños y
niveles de exposición opuestos — el mismo tipo de bug, en otro punto de este
ecosistema, que un identificador compartido entre dos conceptos que
divergen en silencio (el lector ve un nombre familiar y asume que ya sabe de
qué se trata). Un `grep -rn "Token"` en este repo debe encontrar únicamente
cosas con expiración; todo lo demás que autentica sin expirar se llama
`Secret`. Esa es la regla, y es la razón de que `client/consumer.go` declare
`Config.ClientSecret` / `EnvClientSecret = "IAM_CLIENT_SECRET"` — nunca
`ClientToken`.

### Cómo lo resuelve la industria — no es una convención inventada aquí

La distinción viene directo de **OAuth 2.0** (RFC 6749), que es el vocabulario
que este proyecto ya declara seguir (`ARCHITECTURE.md` §6.4, "estilo OAuth2
client credentials"):

- **§2.3 — Client Authentication:** define `client_id`/`client_secret` como la
  credencial con la que un cliente **se autentica ante el servidor de
  autorización**. Es sobre la identidad de la aplicación.
- **§1.4/§1.5 — Access Token / Refresh Token:** define los tokens como la
  credencial que representa **una concesión de acceso** — de duración
  acotada, emitida como resultado de una autenticación exitosa.

Dos secciones separadas, dos palabras separadas, por diseño — no es un detalle
de nomenclatura menor, es la base de todo el modelo de seguridad del RFC. Los
proveedores que implementan client credentials lo replican sin excepción:

| Proveedor | Credencial estática de la app | Credencial dinámica de la sesión/grant |
|---|---|---|
| OAuth 2.0 / OpenID Connect Core | `client_id` + `client_secret` (`client_secret_post`/`client_secret_basic`) | `access_token`, `id_token`, `refresh_token` |
| Google Cloud / Identity Platform | "Client ID" + "Client Secret" del OAuth client | access token / ID token por usuario |
| GitHub Apps | `client_id` + `client_secret` de la App | installation access token / user access token (expiran) |
| Auth0 / Okta | "Client Secret" de la Application registrada | Access Token / ID Token emitido tras el login |
| AWS IAM | Access Key + **Secret** Access Key (de un usuario/rol) | STS session token (temporal) |
| Stripe | **Secret** key (`sk_...`) de la cuenta/integración | claves efímeras de corta duración para SDKs móviles |

El patrón es consistente en los seis: **"secret" es estático y prueba quién
llama; "token" es dinámico y prueba qué se le concedió, por cuánto tiempo.**
`iam` sigue exactamente ese patrón — `IAM_CLIENT_SECRET` es la mitad
"`client_secret`" del par; lo que `FetchAuthzToken` devuelve es la mitad
"`access_token`".

### Regla para este repo, hacia adelante

Ningún símbolo o variable de entorno que represente una credencial
**provista una sola vez y rotada a mano** lleva la palabra `Token` en su
nombre — se llama `Secret`. La palabra `Token` queda reservada exclusivamente
para algo que **este servicio emite y expira**. Si en algún momento hace falta
una credencial de un tercer tipo, se le da un nombre propio antes que forzarla
en una de estas dos categorías.

---

## 2. El acceso al panel se decide por `IAM_ADMIN_EMAILS`, no por RBAC

**Alternativa descartada:** dar de alta un proyecto `iam` en la propia D1,
con un rol `iam:admin`, y exigir ese scope para entrar al panel (dogfooding
del RBAC que el servicio ya expone).

### El motivo

El panel **es** la herramienta que crea proyectos y roles. Exigir un rol
para entrar a la herramienta que crea roles es circular: alguien tiene que
poder entrar *antes* de que exista el primer rol. Esa dependencia se rompe
siempre con un dato de configuración fuera de la base — en este ecosistema,
una variable de entorno, igual que `GOOGLE_CLIENT_ID` o `JWT_SECRET`.

El proyecto `iam` auto-administrado **no elimina** esa necesidad: seguiría
haciendo falta sembrar la lista de correos admin desde algún lado en el
deploy (una variable), y encima añade una fila de proyecto especial, un rol
especial y una comprobación de scope distinta a la del resto de rutas. Más
superficie para el mismo resultado.

`IAM_ADMIN_EMAILS` (lista separada por coma) reusa exactamente el patrón que
`config/bootstrap.go` ya tiene (`EnsureRole(..., emails []string)`): un
correo en la lista, autenticado por la sesión SSO real (Google en prod,
identidad mock en local), es super-admin del panel. Una sola comprobación,
en un solo lugar (`config.IsPanelAdmin`), para todas las rutas `/admin/api/*`.

### Regla para este repo, hacia adelante

El acceso a herramientas de administración que *definen* la política de
autorización se decide con configuración de despliegue, nunca con la misma
política que administran. Si el panel algún día necesita niveles de admin
distintos, se resuelve con más estructura en `IAM_ADMIN_EMAILS` (o una
variable hermana), no metiendo al panel dentro del RBAC que edita.

---

## 3. La corrección la prueban los tests; lo manual es solo la experiencia

**Alternativa descartada:** sembrar en `iam` un proyecto de desarrollo fijo
(p. ej. `misitio` con un `client_secret` conocido) para que un consumidor
pueda hablar con un `iam` local sin pasos manuales, o relajar la
verificación de `client_secret` cuando se detecta entorno de desarrollo.

### El motivo

Las dos variantes introducen una diferencia de comportamiento entre local y
producción — exactamente lo que rompe la confianza en "si funciona en local,
funciona desplegado". Un `iam` que en local no verifica el secreto (o que
trae uno precargado que producción no tiene) esconde el fallo hasta que
alguien despliega.

- **`iam` no conoce a sus consumidores.** `misitio` conoce a `iam` (lo
  consume); `iam` no tiene por qué saber que `misitio` existe. Sembrar
  `misitio` dentro de `iam` invierte esa dirección de dependencia. Un test
  que necesita un proyecto lo **crea** con `config.CreateProject` (o a
  través del panel/API) contra `orm.New(mem.New())`, lo usa, y lo descarta —
  no hay estado que persistir.
- **La verificación de `client_secret` es una sola ruta de código**,
  ejercida por los tests de `tests/` con la misma severidad en local que en
  el Worker. Rotación que invalida el secreto viejo, aislamiento por
  `project_id`, `403` para el no-admin: todo eso es un test, no una
  comprobación a ojo.
- **Lo único que se revisa a mano es la experiencia de uso y el aspecto**
  del panel — porque el gusto y la experiencia no se automatizan. Se levanta
  `web/server.go`, se entra con la identidad mock, y se ajusta lo visual.
  Esta distinción —tests para lo funcional, manual solo para lo visual— es
  regla del ecosistema (ver skill `testing`).

### Regla para este repo, hacia adelante

Ninguna diferencia de comportamiento entre desarrollo y producción para
"facilitar" las pruebas locales. Si algo es incómodo de probar en local, la
respuesta es un test o un fake inyectable, nunca una rama `if dev`.
