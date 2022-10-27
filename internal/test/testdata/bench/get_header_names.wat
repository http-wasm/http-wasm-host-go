(module $get_header_names

  (import "http_handler" "get_header_names" (func $get_header_names
    (param $kind i32)
    (param $buf i32) (param $buf_limit i32)
    (result (; count << 32| len ;) i64)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (global $buf i32 (i32.const 64))
  (global $buf_limit i32 (i32.const 64))

  (func (export "handle_request") (result (; ctx_next ;) i64)
    (call $get_header_names
      (i32.const 0) ;; header_kind_request
      (global.get $buf) (global.get $buf_limit))
    (drop)

    ;; skip any next handler as the benchmark is about get_header_names.
    (return (i64.const 0)))

  ;; handle_response should not be called as handle_request returns zero.
  (func (export "handle_response") (param $reqCtx i32) (param $is_error i32)
    (unreachable))
)

