Good questions — three separate things:

1. What is slog.Duration()?
It's a typed attribute constructor, one of the family alongside slog.String, slog.Int, slog.Float64, slog.Bool, slog.Time. slog.Duration("dur", d) builds a slog.Attr (a key + a typed Value) whose kind is KindDuration, holding a time.Duration. The typed constructors are the safer alternative to the loose "key", value variadic form — they're type-checked at compile time and slightly faster (no boxing/reflection). The rendering difference you hit earlier is the handler's job: JSONHandler emits a KindDuration as raw nanoseconds, TextHandler as "626µs".

2. Should some values be set once with With?
logger.With(attrs...) returns a new logger that carries those attributes and stamps them onto every record it emits. For your current single log line, it buys nothing — you'd just be setting fields for one message. Its value appears when the same logger emits multiple records. So the real pattern isn't "use With inside LogRequests," it's:

Build a request-scoped logger once: reqLog := slog.With("request_id", id, "method", r.Method, "path", r.URL.Path).
Propagate it (via context.Context) into the handlers, so their log lines — a storage error, a slow query — automatically carry the same request_id/method/path without repeating them.
Without propagation, With in the middleware is premature. With propagation, it's exactly right and sets up your "log chains."

3. Per-request unique identifier (the key enabler for #2).
This is a small middleware that runs before logging:

Generate an ID per request. Stdlib-only, matching your existing newID style: crypto/rand → hex (or google/uuid if you want the dependency). Prefer honoring an inbound X-Request-Id header if present (so a caller/proxy can correlate), else generate one.
Stash it in the request context and also echo it back as an X-Request-Id response header (lets clients report the ID when something breaks).
Downstream, retrieve the request-scoped logger (or ID) from context.
Sketch:

Then LogRequests and handlers both call LoggerFrom(r.Context()), so every line in a request shares request_id. Middleware order in main (outer → inner): RequestID → LogRequests → RejectUnsafePaths → mux.

Two idiomatic notes:

Context values should be keyed by an unexported custom type (like ctxKey above), never a bare string, to avoid collisions across packages — go vet's context check enforces this.
Go 1.21+ also has slog.InfoContext(ctx, …) plus handler-level context extraction as an alternative to passing a logger explicitly; passing the logger in context (above) is the simplest and most common approach.
Want me to add the RequestID middleware + LoggerFrom helper, wire the ordering in main, and switch dur to dur_ms while I'm there?