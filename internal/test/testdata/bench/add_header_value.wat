(module $add_header_value

  (import "http-handler" "add_header_value" (func $add_header_value
    (param $kind i32)
    (param $name i32) (param $name_len i32)
    (param $value i32) (param $value_len i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (global $name i32 (i32.const 0))
  (data (i32.const 0) "Set-Cookie")
  (global $name_len i32 (i32.const 10))

  (global $value i32 (i32.const 16))
  (data (i32.const 16) "a=b")
  (global $value_len i32 (i32.const 3))

  (func $handle (export "handle")
    (call $add_header_value
      (i32.const 1) ;; header_kind_response
      (global.get $name) (global.get $name_len)
      (global.get $value) (global.get $value_len)))
)
