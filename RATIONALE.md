# Notable rationale of http-wasm-host-go

## Guest pool

Wasm guests may have static initialization logic that takes significant time,
for example parsing a config file. Because Wasm is not thread-safe, we cannot
use a single instance to handle requests concurrently and instead have a pool
of initialized guests. If guests were used concurrently, runtime code compiled
to wasm, such as stack pointer tracking and garbage collection, could enter an
undefined state and panic/trap.

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

## Guest pinning

As mentioned in the section above, guests (user-defined handlers compiled to
wasm) are used serially. That said, the request lifecycle must be pinned to
the same guest instance (VM). Pinning ensures any state assigned to the guest
controlled request ID during `handle_request` is visible during
`handle_response`. If a random guest in the pool was used for response
handling, not only might the response side miss data, but it may also miss
cleanup of it.

Technically, pinning can occur via any means available on the host: context
propagation, a thread safe map with lookup, or even returning a scoped response
handler. The handler middleware uses an "out context" to implement this, which
hides the underlying guest pool. In net/http using this is simple as you only
need to pass that to the response side. The mosn stream handler has a filter
per request, and the "out context" is stored as a field in that type. This
allows asynchronous handling to use it.
