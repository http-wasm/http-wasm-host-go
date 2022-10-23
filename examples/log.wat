;; This example module is written in WebAssembly Text Format to show the
;; how a handler works and that it is decoupled from other ABI such as WASI.
;; Most users will prefer a higher-level language such as C, Rust or TinyGo.
(module $log

  ;; log_enabled returns 1 if the $level is enabled. This value may be cached
  ;; at request granularity.
  (import "http-handler" "log_enabled" (func $log_enabled
    (param $level i32)
    (result (; 0 or enabled(1) ;) i32)))

  ;; logs a message to the host's logs at the given $level.
  (import "http-handler" "log" (func $log
    (param $level i32)
    (param $buf i32) (param $buf_limit i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))
  (global $message i32 (i32.const 0))
  (data (i32.const 0) "hello world")
  (global $message_len i32 (i32.const 11))

  (func $handle (export "handle")
    ;; We expect debug logging to be disabled. Panic otherwise!
    (if (i32.eq
          (call $log_enabled (i32.const 0)) ;; log_level_debug
          (i32.const 1)) ;; true
        (then unreachable))

    (call $log
      (i32.const 1) ;; log_level_info
      (global.get $message)
      (global.get $message_len)))
)
