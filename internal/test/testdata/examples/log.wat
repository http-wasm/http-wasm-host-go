;; This example module is written in WebAssembly Text Format to show the
;; how a handler works and that it is decoupled from other ABI such as WASI.
;; Most users will prefer a higher-level language such as C, Rust or TinyGo.
(module $log
  ;; get_body writes the body to memory if it exists and isn't larger than
  ;; $buf_limit. The result is the length of the body in bytes.
  (type $get_body (func
    (param $body i32) (param $body_limit i32)
    (result (; len ;) i32)))

  ;; enable_features tries to enable the given features and returns the entire
  ;; feature bitflag supported by the host.
  (import "http-handler" "enable_features" (func $enable_features
    (param $enable_features i64)
    (result (; enabled_features ;) i64)))

  ;; log logs a message to the host's logs.
  (import "http-handler" "log" (func $log
    (param $message i32) (param $message_len i32)))

  ;; get_request_body consumes the body unless $feature_buffer_request is
  ;; enabled.
  (import "http-handler" "get_request_body" (func $get_request_body
    (type $get_body)))

  ;; next dispatches control to the next handler on the host.
  (import "http-handler" "next" (func $next))

  ;; get_response_body requires $feature_buffer_response.
  (import "http-handler" "get_response_body" (func $get_response_body
    (type $get_body)))

  ;; http-wasm guests are required to export "memory", so that imported
  ;; functions like "log" can read memory.
  (memory (export "memory") 1 (; 1 page==64KB ;))

  ;; define a function table for getting a request or response body.
  (table 8 funcref)
  (elem (i32.const 0) $get_request_body)
  (elem (i32.const 1) $get_response_body)

  ;; required_features := feature_buffer_request|feature_buffer_response
  (global $required_features i64 (i64.const 3))

  ;; must_enable_buffering ensures we can inspect request and response bodies
  ;; without interfering with the next handler.
  (func $must_enable_buffering
    (local $enabled_features i64)

    ;; enabled_features := enable_features(required_features)
    (local.set $enabled_features
      (call $enable_features (global.get $required_features)))

    ;; if enabled_features&required_features == 0 { panic }
    (if (i64.eqz (i64.and
          (local.get $enabled_features)
          (global.get $required_features)))
      (then unreachable)))

  (start $must_enable_buffering)

  ;; load constants into memory used for log.
  (global $request_marker i32 (i32.const 0))
  (data (i32.const 0) "request body:")
  (global $request_marker_len i32 (i32.const 13))

  (global $response_marker i32 (i32.const 32))
  (data (i32.const 32) "response body:")
  (global $response_marker_len i32 (i32.const 14))

  ;; body is the memory offset past any initialization data.
  (global $body i32 (i32.const 1024))

  ;; must_log_body logs the body using the given function table index or fails
  ;; if out of memory.
  (func $must_log_body (param $body_fn i32)
    (local $body_limit i32)
    (local $body_len i32)

    ;; set body_limit to the amount of available memory without growing.
    (local.set $body_limit (i32.sub
      (i32.mul (memory.size) (i32.const 65536))
      (global.get $body)))

    ;; body_len = table[body_fn](body, buf_limit)
    (local.set $body_len
      (call_indirect (type $get_body) (global.get $body) (local.get $body_limit) (local.get $body_fn)))

    ;; if body_len > body_limit { panic }
    (if (i32.gt_s (local.get $body_len) (local.get $body_limit))
      (then unreachable)) ;; out of memory

    (call $log
      (global.get $body)
      (local.get $body_len)))

  ;; handle logs the request and response bodies around the "next" handler.
  (func $handle (export "handle")
    ;; This shows interception before the current request is handled.
    (call $log
      (global.get $request_marker)
      (global.get $request_marker_len))

    (call $must_log_body (i32.const 0))

    ;; This handles the request, in whichever way defined by the host.
    (call $next)

    ;; This shows interception after the current request is handled.
    (call $log
      (global.get $response_marker)
      (global.get $response_marker_len))

    (call $must_log_body (i32.const 1)))
)
