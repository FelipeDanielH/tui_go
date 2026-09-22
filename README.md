# TUI Go

TUI Go es un terrario autónomo y observable que vive por completo en la terminal. Plantas, conejos y lobos perciben su entorno local, satisfacen necesidades, se alimentan, huyen, cazan, se reproducen y mueren sin requerir acciones del jugador.

Esta es la **iteración 3**: incorpora cognición limitada, navegación, herencia biológica y una historia poblacional acotada sin convertir la interfaz en el motor del juego.

## Requisitos

- Go 1.25 o posterior
- Una terminal con color y soporte Unicode
- Linux, macOS o Windows

Las únicas dependencias directas son Bubble Tea v2 y Lip Gloss v2 mediante sus módulos modernos `charm.land/*`.

## Instalación y ejecución

```bash
git clone https://github.com/FelipeDanielH/tui_go.git
cd tui_go
go run .
```

Para repetir exactamente una simulación, indica su seed:

```bash
go run . -seed 42
```

El mundo mide 96×48 celdas por defecto y puede superar ampliamente el viewport:

```bash
go run . -seed 42 -width 160 -height 80
```

También existe un modo headless útil para pruebas, profiling y futuros replays:

```bash
go run . -seed 42 -headless 1000
```

## Controles

| Tecla | Acción |
|---|---|
| `WASD`, `HJKL` o flechas | Mover el cursor y desplazar la cámara |
| `Enter` o `I` | Seleccionar/deseleccionar la entidad bajo el cursor |
| `Tab` | Seleccionar y seguir al siguiente animal |
| `C` | Alternar entre resumen e inspector de mente |
| `V` | Mostrar/ocultar ruta, destino y radio de percepción del seleccionado |
| `E` | Abrir/cerrar la vista ECO con poblaciones e historia reciente |
| `Espacio` | Pausar o reanudar |
| `+` / `-` | Duplicar o reducir la velocidad |
| `1`, `2`, `3`, `4` | Velocidad ×1, ×2, ×4 o ×8 |
| `?` | Abrir/cerrar ayuda |
| `Q`, `Esc` | Salir |

El panel de inspección aparece en terminales anchas. El resumen muestra estado físico y acción; la vista **MIND** muestra intención, destino, ruta activa, personalidad, percepción, utilidades y recuerdos recientes. Una entidad seleccionada queda seguida por cámara y cursor hasta deseleccionarla o mover manualmente el cursor. En terminales estrechas se prioriza el mundo; `Tab` sigue permitiendo localizar animales rápidamente.

## Arquitectura

```text
main.go                 configuración, seed y ensamblaje
internal/sim/
  types.go              identidad, necesidades, cognición y acciones
  cognition.go          memoria limitada, personalidad y observación legítima
  pathfinding.go        BFS determinista y acotado sobre terreno transitable
  genetics.go           genoma, fenotipo, herencia biparental y mutación
  world.go              terreno, entidades e índice espacial incremental
  ai.go                 percepción local, Utility AI, objetivos y navegación
  engine.go             ticks, metabolismo, vida y reproducción
internal/ui/
  model.go              input, cámara y reloj visual
  render.go             glyphs, estilos, viewport e inspección
```

`internal/sim` no importa Bubble Tea ni Lip Gloss. El motor avanza únicamente mediante `Engine.Step()` y toda aleatoriedad procede del RNG creado con la seed del mundo. El orden estable de entidades y la ausencia de goroutines en el dominio hacen reproducibles los resultados para una misma seed, configuración y secuencia de pasos.

La UI sólo traduce mensajes de teclado y temporizador a operaciones del motor. La velocidad indica cuántos ticks se ejecutan por pulso visual; el framerate no forma parte de las reglas de simulación. El índice espacial se mantiene **incrementalmente** al crear, mover o eliminar entidades; la percepción consulta exclusivamente celdas dentro del radio de cada animal, sin conocimiento global del mapa.

Los glyphs están separados de `sim.Kind`. Cada celda visual ocupa dos columnas y se normaliza con `lipgloss.Width`, evitando asumir que bytes, runas y columnas de terminal tienen el mismo ancho. Cambiar símbolos no altera ninguna regla del mundo.

## Reglas actuales

- La vegetación crece y se propaga sobre tierra.
- Los animales guardan hasta 12 recuerdos legítimamente percibidos; agua y comida duran hasta 180 ticks, presa y peligro hasta 72, degradando su confianza con la edad.
- Los conejos buscan vegetación y agua recordadas, exploran destinos locales poco visitados y huyen de peligro visto o recientemente recordado.
- Los lobos persiguen una presa visible o su última posición recordada, pero abandonan la búsqueda cuando el recuerdo expira.
- Los objetivos persisten mientras sean válidos; necesidades críticas, peligro, llegada, recurso ausente, memoria caducada o ruta bloqueada los invalidan.
- BFS calcula rutas sólo al cambiar destino, agotarse la ruta o invalidarse, con una ventana mínima de cuatro ticks entre replans. `Speed` controla cadencia de movimiento y nunca permite saltar varias celdas en un tick.
- Cada animal tiene boldness, curiosity, caution y persistence generadas desde la seed, que ajustan las utilidades sin convertirlos en caricaturas.
- Hambre, sed, energía, salud y edad evolucionan en cada tick.
- Animales maduros y saludables pueden reproducirse al encontrarse localmente.
- Necesidades críticas, vejez y depredación causan muerte.
- Las decisiones compiten mediante puntuaciones de utilidad observables como la acción y el objetivo actuales.

## Calidad y tests

```bash
gofmt -w $(find . -name '*.go')
go vet ./...
go test ./...
go build ./...
```

Los tests del motor cubren determinismo por seed, evolución de necesidades, percepción local, selección de acciones, crecimiento y consumo, muerte y reproducción; además validan creación y expiración de memoria, límites de memoria, objetivos, BFS con obstáculos, rutas imposibles, cache/invalidation de rutas, persecución recordada, huida y personalidad. No dependen de la TUI.

Para medir el coste de un tick:

```bash
go test -bench BenchmarkStep -benchmem ./internal/sim
```

## Limitaciones deliberadas de esta iteración

- El terreno usa lagos sencillos; no hay biomas ni generación procedural avanzada.
- La memoria es episódica y pequeña: no hay mapa persistente, memoria social, comunicación ni aprendizaje estadístico.
- BFS usa el terreno del motor para navegar hacia destinos ya percibidos/recordados; todavía no hay mapas de conocimiento parciales ni pathfinding jerárquico para mundos enormes.
- No hay relaciones sociales, genética ni herencia de personalidad.
- La reproducción no modela genética, sexo ni herencia.
- No hay estaciones, enfermedades, construcción, civilizaciones ni decenas de especies.
- Save/load y replay de acciones aún no están implementados; la seed y el modo headless preparan esas fronteras.
- Los eventos históricos sólo viven en una pequeña cola en memoria.

Estas restricciones son intencionales: la siguiente iteración puede evolucionar cada sistema detrás de fronteras ya separadas y testeables.
- Cada animal tiene seis genes: tamaño, velocidad, percepción, metabolismo, resistencia y fertilidad. El fenotipo aplica trade-offs: tamaño/percepción elevan metabolismo, velocidad eleva coste de movimiento y resistencia mejora alimento/longevidad.
- Los hijos combinan ambos genomas, reciben una mutación pequeña acotada y registran generación y ambos IDs parentales. La personalidad se hereda débilmente, pero no es genética física.
- Las poblaciones se muestrean cada 50 ticks en un buffer de 64 muestras; `E` muestra sparklines sin afectar la simulación.
