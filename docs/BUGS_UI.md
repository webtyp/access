# Bugs de UI y mejoras — `webtyp/access`

> Auditoría en vivo sobre `#access.people`, `#access.roles`, `#access.assignments`, `#access.audit` (2026-11-19).

## Bugs encontrados

| # | Sev | Pantalla | Qué se ve en pantalla | Causa raíz verificada |
|---|---|---|---|---|
| AC-1 | 🔴 | `#access.people` | **Panel principal completamente gris/vacío incluso al seleccionar una persona, y botón `+` azul activo.** Seleccionar «Cesar» o «juenifer fuentes» no muestra ningún dato; pulsar `+` cambia el botón a deshacer (`↺`) sin abrir formulario. | `access/definition.go:48-60` define `PersonModel` con `Allow: model.Read` y campos `model.Text()` sin widgets, y `access/view.go:20-26` usa `view.New(...)` que implementa `view.Saver` aun en solo lectura (`crudview.go:1101`). |
| AC-2 | 🔴 | `#access.audit` | **Lista vacía `0 / 0` con panel gris vacío y botón `+` habilitado.** La pantalla es de solo lectura, pero invita a crear un registro con el botón primario `+`. | `access/definition.go:11-24` (`AuditEntryModel`, `Allow: model.Read`) + `access/view.go:47-53` (`view.New` con `Saver` activo) + falta de estado vacío explicativo. |
| AC-3 | 🟡 | `#access.assignments` | **La lista muestra códigos técnicos en inglés (`admin`, `manager`, `practitioner`)** en lugar de los nombres humanos de los roles (`Admin`, `Administración del consultorio`, `Profesional de salud`), mientras que `#access.people` sí muestra los nombres traducidos. | `AssignmentModel` (`access/definition.go:70`) marca `role_code` como `ItemDescription` enviando el código crudo en la etiqueta de fila, mientras `RoleRecordModel` tiene el `name` legible. |
| AC-4 | 🟡 | `#access.roles` | **Rol del sistema `authenticated` aparece en inglés sin traducir** en la lista (`Admin`, `authenticated`, `Administración del consultorio`, ...). | Semilla/rol base en `rbac` o `access` sin clave en `lang.json` o sin traducción aplicada al nombre del rol `authenticated`. |
| AC-5 | 🟡 | `#access.roles`, `#access.assignments` | **El formulario aparece vacío y editable antes de pulsar `+` o seleccionar una fila**, y los botones de papelera/lápiz están visibles con `0 / N` seleccionados. | Comportamiento por defecto de `layout/crudview` (ver bug de framework F8 en el master). |
| AC-6 | ⚪ | `#access.people` | **Badges de rol largos truncados** (`Profesional d...`) y contador `0 / 57` ambiguo cuando hay un ítem seleccionado. | Ancho del badge en `targetlist` + `targetlist__sel-header` cuenta selección múltiple en vez de reflejar selección única. |

## Sugerencias de mejora justificadas

1. **Vista de detalle de solo lectura en `Personas` (`#access.people`):** al seleccionar una persona, mostrar una ficha resumen (nombre, correo/RUT, roles asignados con enlace directo a `#access.assignments` y equipos vinculados) y ocultar `+` cuando `Allow` no incluye `model.Create`. *Justificación:* cumple `ui-consistency` («una lista de solo lectura ofrece acciones con sentido, nunca una pantalla vacía») y resuelve AC-1 sin obligar a saltar a `Asignaciones` para ver los detalles de alguien.
2. **Mostrar el nombre humano del rol en filas de `Asignaciones`:** resolver `role_code → role.name` (tal como ya hace `Personas`). *Justificación:* las personas de administración no tienen por qué memorizar códigos internos en inglés (`practitioner`, `manager`).
3. **Estado vacío accionable en `Auditoría` (`#access.audit`) y ocultar `+`:** mostrar un mensaje claro cuando no hay eventos («Aún no hay movimientos registrados. Aquí aparecerán los cambios de roles y permisos.») y suprimir la barra de creación. *Justificación:* elimina la confusión de ver un botón `+` que no hace nada.
4. **Traducir el rol `authenticated` («Usuario autenticado»):** *Justificación:* regla de `ui-consistency` (cero etiquetas en inglés visibles en pantalla).
