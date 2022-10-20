;; $ wat2wasm --debug-names set_response_header.wat
(module $set_response_header
  (import "http-handler" "set_response_header"
    (func $set_response_header (param i32 i32 i32 i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (global $name i32 (i32.const 0))
  (data (i32.const 0) "Content-Type")
  (global $name_len i32 (i32.const 12))

  (global $value i32 (i32.const 16))
  (data (i32.const 16) "text/plain")
  (global $value_len i32 (i32.const 10))

  (func $handle (export "handle")
    (call $set_response_header
      (global.get $name) (global.get $name_len)
      (global.get $value) (global.get $value_len)))
)
