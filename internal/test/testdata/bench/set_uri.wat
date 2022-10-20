(module $set_uri
  (import "http-handler" "set_uri"
    (func $set_uri (param i32 i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))
  (global $uri i32 (i32.const 0))
  (data (i32.const 0) "/v1.0/hello?name=teddy")
  (global $uri_len i32 (i32.const 22))

  (func $handle (export "handle")
    (call $set_uri
      (global.get $uri)
      (global.get $uri_len)))
)
