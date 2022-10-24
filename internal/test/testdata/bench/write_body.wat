(module $write_body

  (import "http_handler" "write_body" (func $write_body
    (param $kind i32)
    (param $buf i32) (param $buf_len i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (global $body i32 (i32.const 0))
  (data (i32.const 16) "hello world")
  (global $body_len i32 (i32.const 11))

  (func $handle (export "handle")
    (call $write_body
      (i32.const 1) ;; body_kind_response
      (global.get $body) (global.get $body_len))
  )
)
