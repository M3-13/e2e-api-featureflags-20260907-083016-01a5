VERDICT: APPROVED

## Prüfrahmen

Projekttyp: `go-backend` — eine reine REST-API ohne Endnutzer-UI. Daher sind Pflichttexte, Cookie-/Consent-Banner und Anforderungen der Barrierefreiheit für diese Codebasis nicht einschlägig. Relevant sind die DSGVO, soweit personenbezogene Daten verarbeitet oder protokolliert werden, sowie der Cyber Resilience Act, soweit es sich um ein Produkt mit digitalen Elementen handelt. Der EU AI Act ist nicht anwendbar, da keine KI-Funktion erkennbar ist.

---

## 1. DSGVO

### Befund G-01 — `user`-Wert wird als Klartext-Query-Parameter übertragen
**Schweregrad:** mittel  
**Befund:**  
Der Endpunkt `GET /flags/{key}/evaluate?user=alice` verarbeitet eine Nutzerkennung. Der eigene Access-Log in `middleware.go` protokolliert korrekt nur `r.URL.Path` und nicht den Query-String, sodass `user` nicht im Dienst-Log erscheint. Die Übertragung per Query-String bedeutet jedoch, dass vorgelagerte Proxys, Load-Balancer, CDNs oder Browser-Verläufe den Wert mitschneiden können. Die Minimierung im eigenen Code reicht daher allein nicht aus.

**Maßnahme:**  
In `README.md` einen Abschnitt „Datenschutz / Betrieb“ ergänzen:  
> „Der Query-Parameter `user` der Route `GET /flags/{key}/evaluate` darf in vorgelagerten Proxy-, Load-Balancer- oder CDN-Logs nicht gespeichert werden. Entsprechende Logs müssen Query-Strings ausblenden oder die Route darf nur über TLS-terminierte Verbindungen betrieben werden.“

Der bestehende Endpoint bleibt unverändert, damit AC-08/AC-09 nicht gebrochen werden. Eine optionale, kompatible Ergänzung wäre ein alternativer POST-Endpoint mit `user` im Body, ohne den GET-Endpoint zu entfernen.

---

### Befund G-02 — `key` und `description` können beliebige Zeichenketten enthalten
**Schweregrad:** niedrig  
**Befund:**  
`flags_create.go` und `flags_update.go` validieren `key`, `description` und `enabled` nicht auf erlaubte Zeichen oder Länge. Damit könnten Betreiber personenbezogene Daten in einem Flag-Schlüssel oder einer Beschreibung ablegen. Diese Werte werden über die API ausgelesen und in `middleware.go` teilweise im Pfad geloggt.

**Maßnahme:**  
In `flags_create.go` und `flags_update.go` eine Validierung ergänzen, z. B.:
- `key`: maximal 128 Zeichen, erlaubt `^[a-zA-Z0-9._-]+$`
- `description`: maximal 512 Zeichen, keine Pflicht zur inhaltlichen Prüfung, aber README-Hinweis: „Keine personenbezogenen Daten in `key` oder `description` ablegen.“

Dies erhält die bestehende Funktionalität für die in den Tests verwendeten Werte (`dark-mode`, `beta-dash`, `alpha-test`) und verletzt keine Akzeptanzkriterien.

---

### Befund G-03 — Kein TLS im ausgelieferten HTTP-Server
**Schweregrad:** mittel  
**Befund:**  
`main.go` startet ausschließlich `http.ListenAndServe` ohne TLS. Bei einem produktiven Betrieb über ein Netzwerk wären der `user`-Query-Parameter und die Flag-Konfiguration im Klartext lesbar. Dies betrifft Art. 32 DSGVO (Sicherheit der Verarbeitung).

**Maßnahme:**  
In `README.md` verbindlich dokumentieren:  
> „Der Dienst darf nur hinter einer TLS-terminierenden Komponente (Reverse-Proxy, Load-Balancer, API-Gateway) betrieben werden. Direkte Exposition über HTTP ohne TLS ist nicht zulässig.“

Optional in `main.go` eine TLS-Startvariante ergänzen, die über Umgebungsvariablen `TLS_CERT` und `TLS_KEY` aktiviert wird. Die Standard-Startweise (`go run .` ohne externe Abhängigkeiten) bleibt dabei erhalten, damit AC-12 nicht bricht.

---

### Positiv geprüft — Datenminimierung und Logging
**Schweregrad:** keine Beanstandung  
**Befund:**  
- `middleware.go` protokolliert ausschließlich Methode, Pfad, Statuscode und Dauer. Kein Query-String, keine Nutzerkennung.
- `store.go` speichert nur Flag-Datensätze (`key`, `enabled`, `description`, `rollout_percent`). Es gibt keine Persistenz von Nutzerkennungen oder Auswertungsergebnissen.
- `health.go` gibt nur `{"status":"ok"}` zurück und exponiert keine Store-Inhalte.
- Fehlerantworten in `writeError` und `recoveryMiddleware` enthalten keine Stacktraces, Dateipfade oder internen Fehlerstrings.

Die AC-19, AC-20, AC-21 und AC-22 sind im sichtbaren Code erfüllt.

---

## 2. EU Cyber Resilience Act (CRA)

### Befund C-01 — SBOM-/Abhängigkeitsdokumentation nicht sichtbar
**Schweregrad:** niedrig  
**Befund:**  
`go.mod` ist vorhanden und die Anwendung nutzt laut sichtbarem Code ausschließlich die Go-Standardbibliothek. Eine explizite SBOM oder dokumentierte Abhängigkeitsliste ist im gezeigten Stand nicht sichtbar. Für die CRA-Konformität sollte die Abhängigkeitslage nachvollziehbar dokumentiert sein.

**Maßnahme:**  
In `README.md` oder einer separaten `SBOM.md` festhalten:
- Go-Version
- Modulpfad
- Liste der direkten und indirekten Abhängigkeiten (z. B. Ausgabe von `go list -m all`)
- Hinweis: „Keine externen Laufzeitabhängigkeiten; ausschließlich Standardbibliothek.“

---

### Befund C-02 — Sicherheitseigenschaften und Update-/Patch-Prozess nicht dokumentiert
**Schweregrad:** niedrig  
**Befund:**  
Der Code setzt bereits mehrere CRA-relevante Sicherheitsmaßnahmen um: `ReadHeaderTimeout`, `IdleTimeout`, `http.MaxBytesReader` (1 MiB), Recovery-Middleware, keine CORS-Freigabe-Header, keine internen Fehlerdetails. Eine Dokumentation dieser Sicherheitseigenschaften und des Patch-/Update-Prozesses ist im sichtbaren Stand nicht belegt.

**Maßnahme:**  
In `README.md` einen Abschnitt „Security properties“ ergänzen:
- Timeouts: `ReadHeaderTimeout` und `IdleTimeout` jeweils 5 s
- Request-Body-Limit: 1 MiB (`maxBodyBytes` in `middleware.go`)
- Panic-Recovery: `recoveryMiddleware` antwortet mit 500 und JSON-Fehlerobjekt
- Fehlerantworten ohne interne Details
- Keine CORS-Freigabe-Header
- Prozess für Sicherheitsupdates: z. B. „Sicherheitsrelevante Abhängigkeiten werden bei Bekanntwerden von Schwachstellen aktualisiert; Releases werden über Tags versioniert.“

---

### Befund C-03 — Kein Versions-/Build-Endpoint sichtbar
**Schweregrad:** niedrig  
**Befund:**  
Es gibt keinen Endpoint, der Versions- oder Build-Informationen liefert. Für Betrieb, Update-Fähigkeit und Rückverfolgbarkeit ist eine Versionsangabe hilfreich.

**Maßnahme:**  
Optional einen separaten Endpoint `GET /version` ergänzen, der z. B. `{"version":"<version>"}` zurückgibt. Der bestehende `/healthz` sollte unverändert bleiben, da AC-22 dort ausschließlich `{"status":"ok"}` verlangt.

---

## 3. EU AI Act

Nicht anwendbar. Der Dienst enthält keine KI-Funktion; die Rollout-Entscheidung ist ein deterministischer Hash-basierter Algorithmus ohne maschinelles Lernen oder autonome Entscheidungsfindung im Sinne des AI Act.

---

## 4. Pflichttexte & UI

Nicht anwendbar. Es handelt sich um eine reine REST-API ohne Browser-UI, ohne Cookies, ohne Endnutzer-Webseite und ohne Warenkorb/Withdrawal-Flow. Es ergeben sich keine Impressums-, Datenschutzerklärungs-, Cookie- oder Widerrufs-Pflichten aus dem sichtbaren Code selbst.

---

## 5. Barrierefreiheit

Nicht anwendbar. Es existiert keine öffentliche Web-Oberfläche; die API liefert ausschließlich JSON-Antworten.

---

## Fazit

Keine offenen rechtlichen Blocker. Die datenschutzrechtlich relevanten Logging- und Speicherpfade sind sauber umgesetzt. Die verbleibenden Punkte sind Härtungs- und Dokumentationsmaßnahmen, insbesondere TLS-Betrieb, Betreiberhinweise zu Query-Strings sowie CRA-Dokumentation. Sie erfordern keine Änderung, die bestehende Funktionalität oder Akzeptanzkriterien bricht.