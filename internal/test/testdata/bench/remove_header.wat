(module $remove_header
  (import "http_handler" "remove_header"
    (func $remove_header (param i32 i32 i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (global $name i32 (i32.const 0))
  (data (i32.const 0) "Set-Cookie")
  (global $name_len i32 (i32.const 10))

  (func $handle (export "handle")
    (call $remove_header
      (i32.const 1) ;; header_kind_response
      (global.get $name) (global.get $name_len)))
)
