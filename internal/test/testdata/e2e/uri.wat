(module $uri

  (import "http_handler" "get_uri" (func $get_uri
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  (import "http_handler" "set_uri" (func $set_uri
    (param $uri i32) (param $uri_len i32)))

  (import "http_handler" "write_body" (func $write_body
    (param $kind i32)
    (param $buf i32) (param $buf_len i32)))

  (import "http_handler" "next" (func $next))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (global $uri i32 (i32.const 0))
  (data (i32.const 0) "/v1.0/hello?name=teddy")
  (global $uri_len i32 (i32.const 22))

  (global $buf i32 (i32.const 1024))

  ;; handle changes the uri to with the original uri as the request body. Then,
  ;; it dispatches to the next handler.
  (func (export "handle")
    (local $len i32)

    ;; read the uri into memory at offset zero.
    (local.set $len
      (call $get_uri (global.get $buf) (i32.const 1024)))

    ;; delete the URI, which tests setting to "" doesn't crash.
    (call $set_uri (global.get $uri) (i32.const 0))

    ;; change the uri
    (call $set_uri (global.get $uri) (global.get $uri_len))

    ;; write the uri to the request body.
    (call $write_body
      (i32.const 0) ;; body_kind_request
      (global.get $buf) (local.get $len))

    ;; call the next handler which verifies state
    (call $next))
)
