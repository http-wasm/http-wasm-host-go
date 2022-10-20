;; $ wat2wasm --debug-names write_response_body.wat
(module $write_response_body
  (import "http-handler" "write_response_body" (func $write_response_body
    (param $buf i32) (param $buf_len i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (global $body i32 (i32.const 0))
  (data (i32.const 16) "hello world")
  (global $body_len i32 (i32.const 11))

  (func $handle (export "handle")
    (call $write_response_body
      (global.get $body) (global.get $body_len))
  )
)
