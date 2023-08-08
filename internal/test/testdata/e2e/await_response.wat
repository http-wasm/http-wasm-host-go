(module $await_response

(; begin adapter logic

   Below is generic and can convert any normal handler to a single synchronous
   call, provided $handle_request and $handle_response are not exported. ;)

  ;; import $await_response, which blocks until the response is ready.
  (import "http_handler" "await_response" (func $await_response
    (param $ctx_next i64)
    (result (; is_error ;) i32)))

  ;; define a start function that performs a request-response without exports.
  ;; note: this logic is generic and can convert any exported $handle_request/
  ;; $handle_response pair to a synchronous call without exports.
  (func $start
    (local $ctx_next i64)
    (local $is_error i32)
    (local $ctx i32)

    ;; ctxNext := handleRequest()
    (local.set $ctx_next (call $handle_request))

    ;; isError := awaitResponse(ctxNext)
    (local.set $is_error (call $await_response (local.get $ctx_next)))

    ;; ctx = uint32(ctxNext >> 32)
    (local.set $ctx
      (i32.wrap_i64 (i64.shr_u (local.get $ctx_next) (i64.const 32))))

    (call $handle_response (local.get $ctx) (local.get $is_error))
  )

(; end adapter logic ;)

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  ;; reqCtx is the upper 32-bits of the $ctx_next result the host should
  ;; propagate from handle_request to handle_response.
  (global $ctx (export "reqCtx") (mut i32) (i32.const 42))

  ;; handle_request sets the request ID to the global then increments the
  ;; global.
  (func $handle_request (result (; ctx_next ;) i64)
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
  (func $handle_response (param $ctx i32) (param $is_error i32)
    ;; if reqCtx != global.reqCtx - 1 { panic }
    (if (i32.ne
          (local.get $ctx)
          (i32.sub (global.get $ctx) (i32.const 1)))
      (then unreachable))) ;; fail as the host didn't propagate the reqCtx
)
