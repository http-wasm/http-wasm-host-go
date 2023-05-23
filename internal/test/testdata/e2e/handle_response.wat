(module $handle_response

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  ;; reqCtx is the upper 32-bits of the $ctx_next result the host should
  ;; propagate from handle_request to handle_response.
  (global $reqCtx (export "reqCtx") (mut i32) (i32.const 42))

  ;; handle_request sets the request ID to the global then increments the
  ;; global.
  (func (export "handle_request") (result (; ctx_next ;) i64)
    (local $reqCtx i32)

    ;; reqCtx := global.reqCtx
    (local.set $reqCtx (global.get $reqCtx))

    ;; global.reqCtx++
    (global.set $reqCtx (i32.add (global.get $reqCtx) (i32.const 1)))

    ;; return uint64(reqCtx) << 32 | uint64(1)
    (return
      (i64.or
        (i64.shl (i64.extend_i32_u (local.get $reqCtx)) (i64.const 32))
        (i64.const 1))))

  ;; If propagation works, the current request ID should be one less than the
  ;; global.
  (func (export "handle_response") (param $reqCtx i32) (param $is_error i32)
    ;; if reqCtx != global.reqCtx - 1 { panic }
    (if (i32.ne
          (local.get $reqCtx)
          (i32.sub (global.get $reqCtx) (i32.const 1)))
      (then unreachable))) ;; fail as the host didn't propagate the reqCtx
)
