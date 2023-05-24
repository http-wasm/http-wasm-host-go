(module $header_value
  (import "http_handler" "set_header_value" (func $set_header_value
    (param $kind i32)
    (param $name i32) (param $name_len i32)
    (param $value i32) (param $value_len i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (global $name i32 (i32.const 0))
  (data (i32.const 0) "Content-Type")
  (global $name_len i32 (i32.const 12))

  (global $value i32 (i32.const 16))
  (data (i32.const 16) "text/plain")
  (global $value_len i32 (i32.const 10))

  ;; handle_request sets the "Content-Type" to "text/plain". Then, it returns
  ;; non-zero to proceed to the next handler.
  (func (export "handle_request") (result (; ctx_next ;) i64)

    (call $set_header_value
      (i32.const 0) ;; header_kind_request
      (global.get $name) (global.get $name_len)
      (global.get $value) (global.get $value_len))

    ;; call the next handler
    (return (i64.const 1)))

  ;; handle_response is no-op as this is a request-only handler.
  (func (export "handle_response") (param $reqCtx i32) (param $is_error i32))
)
