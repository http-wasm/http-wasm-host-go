;; This example module is written in WebAssembly Text Format to show the
;; how a handler works and that it is decoupled from other ABI such as WASI.
;; Most users will prefer a higher-level language such as C, Rust or TinyGo.
(module $config
  ;; enable_features tries to enable the given features and returns the entire
  ;; feature bitflag supported by the host.
  (import "http-handler" "enable_features" (func $enable_features
    (param $enable_features i64)
    (result (; enabled_features ;) i64)))

  ;; get_config writes configuration from the host to memory if it exists and
  ;; isn't larger than $buf_limit. The result is its length in bytes.
  (import "http-handler" "get_config" (func $get_config
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  ;; next dispatches control to the next handler on the host.
  (import "http-handler" "next" (func $next))

  ;; handle just calls next.
  (func (export "handle") (call $next))

  ;; http-wasm guests are required to export "memory", so that imported
  ;; functions like "get_header" can read memory.
  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (func $must_enable_features
    (local $config_len i32)
    (local $required_features i64)
    (local $enabled_features i64)

    (local.set $config_len
      (call $get_config (i32.const 0) (i32.const 8)))

    ;; if config_len != size_of_uint64le { panic }
    (if (i32.ne (local.get $config_len) (i32.const 8))
      (then unreachable))

    (local.set $required_features (i64.load (i32.const 0)))

    ;; enabled_features := enable_features(required_features)
    (local.set $enabled_features
      (call $enable_features (local.get $required_features)))

    ;; if required_features == 0
    (if (i64.eqz (local.get $required_features))
      ;; if enabled_features != 0 { panic }
      (then (if (i64.ne
          (local.get $enabled_features)
          (i64.const 0))
        (then unreachable)))
      ;; else if enabled_features&required_features == 0 { panic }
      (else (if (i64.eqz (i64.and
          (local.get $enabled_features)
          (local.get $required_features)))
        (then unreachable)))))

  (start $must_enable_features)
)
