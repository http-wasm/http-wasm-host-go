(module $set_status_code
  (import "http_handler" "set_status_code"
    (func $set_status_code (param i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (func (export "handle_request") (result (; ctx_next ;) i64)
    (call $set_status_code (i32.const 404))

    ;; skip any next handler as the benchmark is about set_status_code.
    (return (i64.const 0)))

  ;; handle_response should not be called as handle_request returns zero.
  (func (export "handle_response") (param $reqCtx i32) (param $is_error i32)
    (unreachable))
)
