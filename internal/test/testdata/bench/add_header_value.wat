(module $add_header_value

  (import "http_handler" "add_header_value" (func $add_header_value
    (param $kind i32)
    (param $name i32) (param $name_len i32)
    (param $value i32) (param $value_len i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (global $name i32 (i32.const 0))
  (data (i32.const 0) "Set-Cookie")
  (global $name_len i32 (i32.const 10))

  (global $value i32 (i32.const 16))
  (data (i32.const 16) "a=b")
  (global $value_len i32 (i32.const 3))

  (func (export "handle_request") (result (; ctx_next ;) i64)
    (call $add_header_value
      (i32.const 1) ;; header_kind_response
      (global.get $name) (global.get $name_len)
      (global.get $value) (global.get $value_len))

    ;; skip any next handler as the benchmark is about add_header_value.
    (return (i64.const 0)))

  ;; handle_response should not be called as handle_request returns zero.
  (func (export "handle_response") (param $reqCtx i32) (param $is_error i32)
    (unreachable))
)
