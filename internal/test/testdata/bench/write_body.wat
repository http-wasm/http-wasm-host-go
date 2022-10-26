(module $write_body

  (import "http_handler" "write_body" (func $write_body
    (param $kind i32)
    (param $buf i32) (param $buf_len i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (global $body i32 (i32.const 0))
  (data (i32.const 16) "hello world")
  (global $body_len i32 (i32.const 11))

  (func (export "handle_request") (result (; ctx_next ;) i64)
    (call $write_body
      (i32.const 1) ;; body_kind_response
      (global.get $body) (global.get $body_len))

    ;; skip any next handler as the benchmark is about write_body.
    (return (i64.const 0)))

  ;; handle_response should not be called as handle_request returns zero.
  (func (export "handle_response") (param $reqCtx i32) (param $is_error i32)
    (unreachable))
)
