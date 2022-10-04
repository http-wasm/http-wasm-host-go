;; This example module is written in WebAssembly Text Format to show the
;; how a handler works and that it is decoupled from other ABI such as WASI.
;; Most users will prefer a higher-level language such as C, Rust or TinyGo.
(module $log
  ;; log logs a message to the host's logs.
  (import "http-handler" "log" (func $log
    (param $message i32) (param $message_len i32)))

  ;; next dispatches control to the next handler on the host.
  (import "http-handler" "next" (func $next))

  ;; http-wasm guests are required to export "memory", so that imported
  ;; functions like "log" can read memory.
  (memory (export "memory") 1 (; 1 page==64KB ;))

  ;; load constants into memory used for log.
  (global $before_name i32 (i32.const 0))
  (data (i32.const 0) "before")
  (global $before_name_len i32 (i32.const 6))

  (global $after_name i32 (i32.const 8))
  (data (i32.const 8) "after")
  (global $after_name_len i32 (i32.const 5))

  ;; handle logs the before and after message around the "next" handler.
  (func $handle (export "handle")
    ;; This shows interception before the current request is handled.
    (call $log
      (global.get $before_name)
      (global.get $before_name_len))

    ;; This handles the request, in whichever way defined by the host.
    (call $next)

    ;; This shows interception after the current request is handled.
    (call $log
      (global.get $after_name)
      (global.get $after_name_len))))
