(module $handle_response

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  ;; reqCtx is the upper 32-bits of the $ctx_next result the host should
  ;; propagate from handle_request to handle_response.
  (global $ctx (export "reqCtx") (mut i32) (i32.const 42))

  ;; handle_request sets the request ID to the global then increments the
  ;; global.
  (func (export "handle_request") (result (; ctx_next ;) i64)
    (local $ctx i32)

    ;; reqCtx := global.reqCtx
    (local.set $ctx (global.get $ctx))

    ;; global.reqCtx++
    (global.set $ctx (i32.add (global.get $ctx) (i32.const 1)))

    ;; return uint64(reqCtx) << 32 | uint64(1)
    (return
      (i64.or
        (i64.shl (i64.extend_i32_u (local.get $ctx)) (i64.const 32))
        (i64.const 1))))

  ;; If propagation works, the current request ID should be one less than the
  ;; global.
  (func (export "handle_response") (param $ctx i32) (param $is_error i32)
    ;; if reqCtx != global.reqCtx - 1 { panic }
    (if (i32.ne
          (local.get $ctx)
          (i32.sub (global.get $ctx) (i32.const 1)))
      (then unreachable))) ;; fail as the host didn't propagate the reqCtx
)
