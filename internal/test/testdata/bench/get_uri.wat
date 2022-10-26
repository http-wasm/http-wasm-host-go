(module $get_uri

  (import "http_handler" "get_uri"
    (func $get_uri (param i32 i32) (result i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))
  (global $buf i32 (i32.const 0))
  (global $buf_limit i32 (i32.const 64))

  (func (export "handle_request") (result (; ctx_next ;) i64)
    (call $get_uri
      (global.get $buf)
      (global.get $buf_limit))
    (drop)

    ;; skip any next handler as the benchmark is about get_uri.
    (return (i64.const 0)))

  ;; handle_response should not be called as handle_request returns zero.
  (func (export "handle_response") (param $reqCtx i32) (param $is_error i32)
    (unreachable))
)
