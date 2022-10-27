(module $set_uri
  (import "http_handler" "set_uri"
    (func $set_uri (param i32 i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))
  (global $uri i32 (i32.const 0))
  (data (i32.const 0) "/v1.0/hello?name=teddy")
  (global $uri_len i32 (i32.const 22))

  (func (export "handle_request") (result (; ctx_next ;) i64)
    (call $set_uri
      (global.get $uri)
      (global.get $uri_len))

    ;; skip any next handler as the benchmark is about set_uri.
    (return (i64.const 0)))

  ;; handle_response should not be called as handle_request returns zero.
  (func (export "handle_response") (param $reqCtx i32) (param $is_error i32)
    (unreachable))
)
