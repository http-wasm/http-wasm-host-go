;; This example module is written in WebAssembly Text Format to show the
;; how a handler works when the guest compiler doesn't support function
;; exports, such as GOOS=wasip1 in Go 1.21.
(module $log_once

  ;; log_enabled returns 1 if the $level is enabled. This value may be cached
  ;; at request granularity.
  (import "http_handler" "log_enabled" (func $log_enabled
    (param $level i32)
    (result (; 0 or enabled(1) ;) i32)))

  ;; logs a message to the host's logs at the given $level.
  (import "http_handler" "log" (func $log
    (param $level i32)
    (param $buf i32) (param $buf_limit i32)))

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
  (global $message i32 (i32.const 0))
  (data (i32.const 0) "hello world")
  (global $message_len i32 (i32.const 11))

  (func $handle_request (result (; ctx_next ;) i64)
    ;; We expect debug logging to be disabled. Panic otherwise!
    (if (i32.eq
          (call $log_enabled (i32.const -1)) ;; log_level_debug
          (i32.const 1)) ;; true
        (then unreachable))

    (call $log
      (i32.const 0) ;; log_level_info
      (global.get $message)
      (global.get $message_len))

    ;; uint32(ctx_next) == 1 means proceed to the next handler on the host.
    (return (i64.const 1)))

  ;; handle_response is no-op as this is a request-only handler.
  (func $handle_response (param $reqCtx i32) (param $is_error i32))
)
