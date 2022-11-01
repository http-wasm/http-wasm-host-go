# Notable rationale of http-wasm-host-go

## Guest pool

Wasm guests may have static initialization logic that takes significant time,
for example parsing a config file. Because Wasm is not thread-safe, we cannot
use a single instance to handle requests concurrently and instead have a pool
of initialized guests.

The pool size is uncapped - any pool size cap would be a concurrency limit on
the request handler. Production HTTP servers all have a native concept of
concurrency limiting, so it would be redundant to have another way to limit
concurrency here.

It is understood that guests use more memory than the same logic in native
code. For example, some compilers default to a minimum of 16MB, and there is
also overhead for the VM running the guest. This can imply running into a
resource constraint faster than the same logic in native code. However, it
is remains a better choice to address this with your HTTP server's
concurrency limit mechanism.
