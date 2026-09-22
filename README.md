# TUI Go

TUI Go es un terrario autónomo y observable que vive por completo en la terminal. Plantas, conejos y lobos perciben su entorno local, satisfacen necesidades, se alimentan, huyen, cazan, se reproducen y mueren sin requerir acciones del jugador.

Esta es la **iteración 1**: un vertical slice pequeño y funcional, diseñado como base para ampliar la simulación sin convertir la interfaz en el motor del juego.

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
| `Espacio` | Pausar o reanudar |
| `+` / `-` | Duplicar o reducir la velocidad |
| `1`, `2`, `3`, `4` | Velocidad ×1, ×2, ×4 o ×8 |
| `?` | Abrir/cerrar ayuda |
| `Q`, `Esc` | Salir |

El panel de inspección aparece en terminales anchas. Muestra ID, especie, edad, salud, hambre, sed, energía, acción y objetivo. Una entidad seleccionada queda seguida por cámara y cursor hasta deseleccionarla o mover manualmente el cursor. En terminales estrechas se prioriza el mundo; `Tab` sigue permitiendo localizar animales rápidamente.

## Arquitectura

```text
main.go                 configuración, seed y ensamblaje
internal/sim/
  types.go              identidad, necesidades, rasgos y acciones
  world.go              terreno, entidades e índice espacial
  ai.go                 percepción local y Utility AI
  engine.go             ticks, metabolismo, vida y reproducción
internal/ui/
  model.go              input, cámara y reloj visual
  render.go             glyphs, estilos, viewport e inspección
```

`internal/sim` no importa Bubble Tea ni Lip Gloss. El motor avanza únicamente mediante `Engine.Step()` y toda aleatoriedad procede del RNG creado con la seed del mundo. El orden estable de entidades y la ausencia de goroutines en el dominio hacen reproducibles los resultados para una misma seed, configuración y secuencia de pasos.

La UI sólo traduce mensajes de teclado y temporizador a operaciones del motor. La velocidad indica cuántos ticks se ejecutan por pulso visual; el framerate no forma parte de las reglas de simulación. El índice espacial se reconstruye por tick y la percepción consulta exclusivamente celdas dentro del radio de cada animal, sin conocimiento global del mapa.

Los glyphs están separados de `sim.Kind`. Cada celda visual ocupa dos columnas y se normaliza con `lipgloss.Width`, evitando asumir que bytes, runas y columnas de terminal tienen el mismo ancho. Cambiar símbolos no altera ninguna regla del mundo.

## Reglas actuales

- La vegetación crece y se propaga sobre tierra.
- Los conejos buscan vegetación y agua, descansan y huyen de lobos visibles.
- Los lobos buscan agua y persiguen conejos según su hambre.
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

Los tests del motor cubren determinismo por seed, evolución de necesidades, percepción local, selección de acciones, crecimiento y consumo, muerte y reproducción. No dependen de la TUI.

Para medir el coste de un tick:

```bash
go test -bench BenchmarkStep -benchmem ./internal/sim
```

## Limitaciones deliberadas de esta iteración

- El terreno usa lagos sencillos; no hay biomas ni generación procedural avanzada.
- La IA no tiene memoria, personalidad, relaciones sociales ni planificación de rutas.
- Los animales sortean obstáculos mediante movimientos locales; no existe pathfinding global.
- La reproducción no modela genética, sexo ni herencia.
- No hay estaciones, enfermedades, construcción, civilizaciones ni decenas de especies.
- Save/load y replay de acciones aún no están implementados; la seed y el modo headless preparan esas fronteras.
- Los eventos históricos sólo viven en una pequeña cola en memoria.

Estas restricciones son intencionales: la siguiente iteración puede evolucionar cada sistema detrás de fronteras ya separadas y testeables.
