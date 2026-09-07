# Feature-Flag-Service

Ein Feature-Flag-Service als REST-API in Go. Flags werden thread-sicher im
Speicher gehalten (`sync.RWMutex` + `map[string]Flag`); unterstützt werden
Anlegen, Auflisten, Abrufen, Ändern, Löschen sowie deterministische
Rollout-Entscheidungen pro Nutzer. Der Dienst nutzt ausschließlich die
Go-Standardbibliothek (`net/http`) und kommt ohne externe Abhängigkeiten aus.

## Tech Stack

- **Sprache**: Go (Package `main`, flaches Layout)
- **HTTP**: `net/http` (kein externes Web-Framework)
- **Storage**: In-Memory mit `sync.RWMutex`
- **Tests**: `httptest` aus der Standardbibliothek

## Installation

Keine externen Abhängigkeiten. Go 1.22 oder neuer ist ausreichend:

```bash
go version
```

## Entwicklung / Start

```bash
go run .
```

Der Server lauscht auf dem Port aus der Umgebungsvariable `PORT`
(Standard: `8080`):

```bash
# Windows (PowerShell)
$env:PORT="9090"; go run .
# Linux / macOS
PORT=9090 go run .
```

## Build für Produktion

```bash
go build ./...
```

## Tests

```bash
go test ./...
```

## Endpunkte

| Methode | Pfad                      | Beschreibung                          |
| ------- | ------------------------- | ------------------------------------- |
| `POST`  | `/flags`                  | Flag anlegen                          |
| `GET`   | `/flags`                  | Alle Flags auflisten                  |
| `GET`   | `/flags/{key}`            | Einzelnes Flag abrufen                |
| `PUT`   | `/flags/{key}`            | Flag ändern (Partielles Update)       |
| `DELETE`| `/flags/{key}`            | Flag löschen                          |
| `GET`   | `/flags/{key}/evaluate`   | Rollout-Entscheidung (`?user=...`)    |
| `GET`   | `/healthz`                | Health-Check (`{"status":"ok"}`)      |

Fehlerantworten verwenden durchgängig das JSON-Format `{"error": "<message>"}`
mit `application/json`; Stacktraces oder interne Fehlerstrings werden nie an
den Client ausgeliefert.

## Features

- In-Memory-Store mit `sync.RWMutex` (thread-sicher)
- Deterministische Rollout-Entscheidung pro Nutzer (FNV-1a-basiert)
- Zugriffs-Logging (Methode, Pfad ohne Query-String, Status, Dauer)
- Recovery-Middleware (Panics → `500` mit JSON-Fehlerobjekt)
- Request-Body-Limit (1 MiB) und Content-Type-Prüfung
- HTTP-Server-Timeouts (`ReadHeaderTimeout`, `IdleTimeout` = 5 s)
- Kein CORS-Header (kein Cross-Origin-Zugriff)
