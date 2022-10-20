(module $get_uri

  (import "http-handler" "get_uri"
    (func $get_uri (param i32 i32) (result i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))
  (global $buf i32 (i32.const 0))
  (global $buf_limit i32 (i32.const 64))

  (func $handle (export "handle")
    (call $get_uri
      (global.get $buf)
      (global.get $buf_limit))
    (drop))
)
