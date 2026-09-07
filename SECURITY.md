VERDICT: CHANGES_REQUESTED

## Sicherheitsbericht

### Hinweis zur Scanner-Ausgabe
Für den Projekttyp `go-backend` wurde kein anwendbarer Security-Scanner ausgeführt. Eine automatische Dependency- oder Schwachstellenanalyse liegt daher nicht vor. `go.mod` ist in der Dateiliste enthalten, wurde aber nicht im Inhalt gezeigt; laut Sprint-Spec wird ausschließlich die Go-Standardbibliothek verwendet. Daraus leite ich keinen Befund ab.

### Umsetzungsstand der expliziten Security-Anforderungen
Die in den ACs geforderten Härtungen sind weitgehend vorhanden: Body-Limit über `http.MaxBytesReader`, `ReadHeaderTimeout`/`IdleTimeout`, keine CORS-Freigabe-Header, Recovery-Middleware, generische JSON-Fehlerobjekte, Logging ohne Query-String bei `/evaluate`, Health-Endpoint ohne Store-Inhalte. Es verbleiben aber sicherheitsrelevante Lücken bzw. Härtungsbedarf.

---

## Befunde

### M1 — Fehlende Authentifizierung und Autorisierung
**Schweregrad:** mittel (hoch, falls der Dienst öffentlich erreichbar ist)

**Betroffene Stellen:** `main.go` (`newHandler` und alle registrierten Routen), `flags_create.go`, `flags_update.go`, `flags_delete.go`

**Beschreibung:** Alle Endpunkte sind ohne Authentifizierung oder Autorisierung erreichbar. Ein Angreifer mit Netzwerkzugriff kann Flags anlegen, ändern, löschen und Rollout-Prozentsätze manipulieren. Es gibt keine schreibgeschützte oder lesende Zugriffstrennung.

**Konkreter Fix:** Schreib- und Leserouten hinter eine Auth-Middleware legen (z. B. Bearer-Token, API-Key oder mTLS) oder den Dienst zwingend hinter einem authentifizierenden Reverse-Proxy/API-Gateway betreiben. Mindestens für `POST /flags`, `PUT /flags/{key}` und `DELETE /flags/{key}` sollte ein Schreib-Token/Scope verlangt werden, für `GET /flags`/`GET /flags/{key}` optional ein Lese-Token/Scope.

---

### M2 — Transport unverschlüsselt (reines HTTP)
**Schweregrad:** mittel

**Betroffene Stelle:** `main.go`, `server.ListenAndServe()` auf `":" + port`

**Beschreibung:** Der Dienst lauscht unverschlüsselt auf HTTP. Flag-Konfigurationen, Antworten und Anfragen können im Übertragungsnetz mitgelesen oder manipuliert werden, sofern der Dienst nicht bereits hinter einem TLS-Terminierungspunkt betrieben wird.

**Konkreter Fix:** TLS entweder direkt im Dienst mit `http.Server` und `ListenAndServeTLS` oder durch einen vorgeschalteten Reverse-Proxy/Service-Mesh erzwingen. Bei direktem TLS Zertifikatspfade als Konfiguration einlesen, nicht hartkodieren.

---

### M3 — Log-Injection über den Request-Pfad
**Schweregrad:** niedrig

**Betroffene Stelle:** `middleware.go`, Funktion `loggingMiddleware`, Zeile `log.Printf("%s %s %d %s", r.Method, r.URL.Path, status, time.Since(start))`

**Beschreibung:** `r.URL.Path` wird unformatiert in das Log geschrieben. Ein Angreifer kann über percent-kodierte Zeilenumbrüche (z. B. `%0A`) im Request-Pfad zusätzliche Logzeilen erzeugen und so Log-Analysen, Alerting oder Audit-Trails verfälschen.

**Konkreter Fix:** Pfad vor der Ausgabe sanitieren oder mit `%q` loggen:

```go
log.Printf("%s %q %d %s", r.Method, r.URL.Path, status, time.Since(start))
```

Alternativ Steuerzeichen wie `\n`, `\r`, `\t` aus `r.URL.Path` entfernen.

---

### M4 — Fehlende `ReadTimeout`-/`WriteTimeout`-Begrenzung
**Schweregrad:** niedrig

**Betroffene Stelle:** `main.go`, Konfiguration des `http.Server`

**Beschreibung:** `ReadHeaderTimeout` und `IdleTimeout` sind gesetzt, aber `ReadTimeout` und `WriteTimeout` fehlen. Ein Client, der die Header schnell sendet, kann den Body sehr langsam übertragen und dadurch eine Verbindung über lange Zeit belegen. Ebenso fehlt eine Begrenzung der Antwortdauer.

**Konkreter Fix:** Zusätzlich setzen, z. B.:

```go
ReadTimeout:  5 * time.Second, // oder knapp oberhalb der erwarteten 1-MiB-Latenz
WriteTimeout: 10 * time.Second,
```

Die Werte müssen zur maximalen Body-Größe und zur Netzumgebung passen; für eine interne API mit 1 MiB Limit sind 5–15 Sekunden realistisch.

---

### M5 — JSON-Dekodierung härten
**Schweregrad:** niedrig

**Betroffene Stelle:** `middleware.go`, `DecodeJSON`

**Beschreibung:**
- Der Content-Type wird exakt mit `"application/json"` verglichen. Gültige Angaben wie `application/json; charset=utf-8` werden fälschlich mit 415 abgelehnt.
- Nach dem ersten erfolgreichen `Decode` wird nicht geprüft, ob weitere Bytes im Body folgen. Eingaben wie `{"key":"x"}garbage` werden als gültig akzeptiert, sofern der Content-Type exakt passt.

**Konkreter Fix:**
- Content-Type mittels `mime.ParseMediaType` auswerten und nur den Medientyp `application/json` zulassen.
- Nach der ersten Dekodierung das Ende des Bodys prüfen, z. B.:

```go
dec := json.NewDecoder(r.Body)
if err := dec.Decode(dst); err != nil { ... }
if err := dec.Decode(&struct{}{}); err != io.EOF {
    writeError(w, http.StatusBadRequest, "unexpected trailing data")
    return errors.New("trailing data")
}
```

---

### M6 — Recovery bei bereits teilweise geschriebener Antwort
**Schweregrad:** niedrig

**Betroffene Stelle:** `middleware.go`, `recoveryMiddleware`

**Beschreibung:** Wenn ein Handler nach dem Schreiben von Statuscode oder partiellem Body panikt, ruft die Recovery-Middleware erneut `writeError` auf. Das führt zu zusätzlichen Header-Schreibvorgängen und möglicherweise inkonsistenten Antworten, insbesondere wenn bereits `WriteHeader` gesendet wurde.

**Konkreter Fix:** Einen ResponseWriter verwenden, der den Header-Status trackt. Falls `wroteHeader` bereits `true` ist, nicht erneut `writeError` aufrufen, sondern z. B. `panic(http.ErrAbortHandler)` auslösen, damit die Verbindung sauber abgebrochen wird.

---

### M7 — Flag-Key-Validierung zu schwach
**Schweregrad:** niedrig

**Betroffene Stelle:** `flags_create.go`, `handleCreateFlag`

**Beschreibung:** Der Flag-Key wird nur auf `""` geprüft. Steuerzeichen, Leerzeichen oder beliebige Unicode-Zeichen sind möglich. Das kann unerwartete Log-/JSON-Ausgaben begünstigen und die API-Oberfläche inkonsistent machen.

**Konkreter Fix:** Erlaubte Zeichen und Länge einschränken, z. B.:

```go
var flagKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,128}$`)
```

Bei Verstoß `400` mit generischem Fehlerobjekt (`"invalid key"`) zurückgeben. Die vorhandenen Beispiel-Keys (`dark-mode`, `beta-dash`, `alpha-test`) bleiben damit gültig.

---

## Gesamtbewertung
Es wurden keine kritischen oder hohen Schwachstellen wie hartkodierte Secrets, Injection/RCE, bekannte ausgenutzte CVEs oder unbeabsichtigte PII-Leaks im sichtbaren Code gefunden. Wegen der mittleren Befunde (fehlende Authentifizierung, unverschlüsselter Transport) und des Härtungsbedarfs wird die Auslieferung derzeit nicht freigegeben; nach Umsetzung der genannten Maßnahmen ist eine erneute Prüfung sinnvoll.