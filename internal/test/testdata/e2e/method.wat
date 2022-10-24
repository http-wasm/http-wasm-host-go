(module $method

  (import "http_handler" "get_method" (func $get_method
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  (import "http_handler" "set_method" (func $set_method
    (param $method i32) (param $method_len i32)))

  (import "http_handler" "write_body" (func $write_body
    (param $kind i32)
    (param $buf i32) (param $buf_len i32)))

  (import "http_handler" "next" (func $next))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (global $post i32 (i32.const 0))
  (data (i32.const 0) "POST")
  (global $post_len i32 (i32.const 4))

  (global $buf i32 (i32.const 1024))

  ;; handle changes the method to POST with the original method as the request
  ;; body. Then, it dispatches to the next handler.
  (func (export "handle")
    (local $len i32)

    ;; read the method into memory at offset zero.
    (local.set $len
      (call $get_method (global.get $buf) (i32.const 1024)))

    ;; change the method to POST
    (call $set_method (global.get $post) (global.get $post_len))

    ;; write the method to the request body.
    (call $write_body
      (i32.const 0) ;; body_kind_request
      (global.get $buf) (local.get $len))

    ;; call the next handler which verifies state
    (call $next))
)
