(module $get_header_values

  (import "http-handler" "get_header_values" (func $get_header_values
    (param $kind i32)
    (param $name i32) (param $name_len i32)
    (param $buf i32) (param $buf_limit i32)
    (result (; count << 32| len ;) i64)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (global $name i32 (i32.const 0))
  (data (i32.const 0) "Accept")
  (global $name_len i32 (i32.const 6))

  (global $buf i32 (i32.const 64))
  (global $buf_limit i32 (i32.const 64))

  (func $handle (export "handle")
    (call $get_header_values
      (i32.const 0) ;; header_kind_request
      (global.get $name) (global.get $name_len)
      (global.get $buf) (global.get $buf_limit))
    (drop))
)
