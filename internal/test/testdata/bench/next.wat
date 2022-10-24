(module $next

  (import "http_handler" "next" (func $next))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (func $handle (export "handle") (call $next))
)
