(module $handle_response

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  ;; expectedReqCtx is the upper 32-bits of the $ctx_next result the host
  ;; should propagate from handle_request to handle_response.
  (global $expectedReqCtx i32 (i32.const 43))

  (func (export "handle_request") (result (; ctx_next ;) i64)
    ;; return uint64(expectedReqCtx) << 32 | uint64(1)
    (return
      (i64.or
        (i64.shl (i64.extend_i32_u (global.get $expectedReqCtx)) (i64.const 32))
        (i64.const 1))))

  ;; handle_response should not be called as handle_request returns zero.
  (func (export "handle_response") (param $reqCtx i32) (param $is_error i32)
    ;; if reqCtx != expectedReqCtx { panic }
    (if (i32.ne
          (local.get $reqCtx)
          (global.get $expectedReqCtx))
      (then unreachable))) ;; fail as the host didn't propgate the reqCtx
)
