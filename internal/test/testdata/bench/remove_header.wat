(module $remove_header
  (import "http_handler" "remove_header"
    (func $remove_header (param i32 i32 i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (global $name i32 (i32.const 0))
  (data (i32.const 0) "Set-Cookie")
  (global $name_len i32 (i32.const 10))

  (func (export "handle_request") (result (; ctx_next ;) i64)
    (call $remove_header
      (i32.const 1) ;; header_kind_response
      (global.get $name) (global.get $name_len))

    ;; skip any next handler as the benchmark is about remove_header.
    (return (i64.const 0)))

  ;; handle_response should not be called as handle_request returns zero.
  (func (export "handle_response") (param $reqCtx i32) (param $is_error i32)
    (unreachable))
)
