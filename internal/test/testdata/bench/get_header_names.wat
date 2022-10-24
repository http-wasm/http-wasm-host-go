(module $get_header_names

  (import "http_handler" "get_header_names" (func $get_header_names
    (param $kind i32)
    (param $buf i32) (param $buf_limit i32)
    (result (; count << 32| len ;) i64)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (global $buf i32 (i32.const 64))
  (global $buf_limit i32 (i32.const 64))

  (func $handle (export "handle")
    (call $get_header_names
      (i32.const 0) ;; header_kind_request
      (global.get $buf) (global.get $buf_limit))
    (drop))
)
