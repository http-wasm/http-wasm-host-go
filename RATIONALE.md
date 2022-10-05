# Notable rationale of http-wasm-host-go

## Guest pool

Wasm guests may have static initialization logic that takes significant time,
for example parsing a config file. Because Wasm is not thread-safe, we cannot
use a single instance to handle requests concurrently and instead have a pool
of initialized guests.

The pool size is uncapped - any pool size cap would be a concurrency limit on
the request handler. Production HTTP servers all have a native concept of
concurrency limiting, so it would be redundant to have another way to limit
concurrency here. Memory used by the middleware is similar to end-user business
logic, and high memory usage in either would be handled the same way, by the
HTTP server's concurrency limit mechanism.
