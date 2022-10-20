;; $ wat2wasm --debug-names get_request_header.wat
(module $get_request_header
  (import "http-handler" "get_request_header"
    (func $get_request_header (param i32 i32 i32 i32) (result i64)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (global $name i32 (i32.const 0))
  (data (i32.const 0) "Accept")
  (global $name_len i32 (i32.const 6))

  (global $buf i32 (i32.const 64))
  (global $buf_limit i32 (i32.const 64))

  (func $handle (export "handle")
    (call $get_request_header
      (global.get $name) (global.get $name_len)
      (global.get $buf) (global.get $buf_limit))
    (drop))
)
