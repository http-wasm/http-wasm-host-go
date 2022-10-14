;; This example module is written in WebAssembly Text Format to show the
;; how a handler works and that it is decoupled from other ABI such as WASI.
;; Most users will prefer a higher-level language such as C, Rust or TinyGo.
(module $redact
  ;; get_body writes the body to memory if it exists and isn't larger than
  ;; $buf_limit. The result is the length of the body in bytes.
  (type $get_body (func
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

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

  ;; get_request_body consumes the body unless $feature_buffer_request is
  ;; enabled.
  (import "http-handler" "get_request_body" (func $get_request_body
    (type $get_body)))

  ;; set_request_body overwrites the request body with a value read from memory.
  (import "http-handler" "set_request_body" (func $set_request_body
    (param $body i32)
    (param $len i32)))

  ;; next dispatches control to the next handler on the host.
  (import "http-handler" "next" (func $next))

  ;; get_response_body requires $feature_buffer_response.
  (import "http-handler" "get_response_body" (func $get_response_body
    (type $get_body)))

  ;; set_response_body overwrites the response body with a value read from memory.
  (import "http-handler" "set_response_body" (func $set_response_body
    (param $body i32)
    (param $len i32)))

  ;; http-wasm guests are required to export "memory", so that imported
  ;; functions like $get_response_body can read memory.
  (memory (export "memory") 1 (; 1 page==64KB ;))

  ;; define a function table for getting a request or response body.
  (table 8 funcref)
  (elem (i32.const 0) $get_request_body)
  (elem (i32.const 1) $get_response_body)

  ;; body is the memory offset past any initialization data.
  (global $body i32 (i32.const 1024))

  (global $secret i32 (i32.const 0))
  ;; $secret_len is mutable as it is initialized during start.
  (global $secret_len (mut i32) (i32.const 0))

  ;; must_read_secret ensures there's a non-zero length secret configured.
  (func $must_read_secret
    (local $config_len i32)

    (local.set $config_len
      (call $get_config (global.get $secret) (global.get $body)))

    ;; if config_len > body { panic }
    (if (i32.gt_u (local.get $config_len) (global.get $body))
      (then unreachable))

    ;; secret_len = config_len
    (global.set $secret_len (local.get $config_len))

    ;; if secret_len == 0 { panic }
    (if (i32.eqz (global.get $secret_len))
      (then unreachable)))

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

  (start $main)
  (func $main
    (call $must_enable_buffering)
    (call $must_read_secret))

  ;; must_get_body returns the length of the body using the given function
  ;; table index or fails if out of memory.
  (func $must_get_body (param $body_fn i32) (result (; len ;) i32)
    (local $limit i32)
    (local $len i32)

    ;; set limit to the amount of available memory without growing.
    (local.set $limit (i32.sub
      (i32.mul (memory.size) (i32.const 65536))
      (global.get $body)))

    ;; len = table[body_fn](body, buf_limit)
    (local.set $len
      (call_indirect (type $get_body) (global.get $body) (local.get $limit) (local.get $body_fn)))

    ;; if len > limit { panic }
    (if (i32.gt_u (local.get $len) (local.get $limit))
      (then unreachable)) ;; out of memory

    (local.get $len))

  ;; handle implements a simple HTTP router.
  (func $handle (export "handle")
    (local $len i32)

    ;; load the request body from the upstream handler into memory.
    (local.set $len
      (call $must_get_body (i32.const 0)))

    ;; if redaction affected the copy of the request in memory...
    (if (call $redact (global.get $body) (local.get $len))
      (then ;; overwrite the request body on the host with the redacted one.
        (call $set_request_body (global.get $body) (local.get $len))))

    (call $next) ;; dispatch with $feature_buffer_response enabled.

    ;; load the response body from the downstream handler into memory.
    (local.set $len
      (call $must_get_body (i32.const 1)))

    ;; if redaction affected the copy of the response in memory...
    (if (call $redact (global.get $body) (local.get $len))
      (then ;; overwrite the response body on the host with the redacted one.
        (call $set_response_body (global.get $body) (local.get $len)))))

  ;; redact inline replaces any secrets in the memory region with hashes (#).
  (func $redact (param $ptr i32) (param $len i32) (result (; redacted ;) i32)
    (local $redacted i32)

    (if (i32.eqz (call $can_redact (local.get $len)))
      (then (return (i32.const 0)))) ;; can't redact

    (loop $redacting
      ;; if mem[i:secret_len] == secret
      (if (call $memeq (local.get $ptr) (global.get $secret) (global.get $secret_len))
        (then ;; redact by overwriting the region with hashes (#)
          (local.set $redacted (i32.const 1))
          (memory.fill
            (local.get $ptr)
            (i32.const 35) ;; # in ASCII
            (global.get $secret_len))))

      (local.set $ptr (i32.add (local.get $ptr) (i32.const 1))) ;; ptr++
      (local.set $len (i32.sub (local.get $len) (i32.const 1))) ;; $len--

      ;; if can_redact(len) { continue } else { break }
      (br_if $redacting (call $can_redact (local.get $len))))

    ;; return whether the memory changed due to redaction
    (local.get $redacted))

  ;; can_redact ensures the current pointer can be compared to the secret.
  (func $can_redact (param $len i32) (result (; ok ;) i32)
    (i32.and
      (i32.gt_u (global.get $secret_len) (local.get $len)
      (i32.gt_u (local.get $len) (i32.const 0)))))

  ;; memeq is like memcmp except it returns 0 (ne) or 1 (eq)
  (func $memeq (param $ptr1 i32) (param $ptr2 i32) (param $len i32) (result i32)
    (local $i1 i32)
    (local $i2 i32)
    (local.set $i1 (local.get $ptr1)) ;; i1 := ptr1
    (local.set $i2 (local.get $ptr2)) ;; i2 := ptr1

    (loop $len_gt_zero
      ;; if mem[i1] != mem[i2]
      (if (i32.ne (i32.load8_u (local.get $i1)) (i32.load8_u (local.get $i2)))
        (then (return (i32.const 0)))) ;; return 0

      (local.set $i1  (i32.add (local.get $i1)  (i32.const 1))) ;; i1++
      (local.set $i2  (i32.add (local.get $i2)  (i32.const 1))) ;; i2++
      (local.set $len (i32.sub (local.get $len) (i32.const 1))) ;; $len--

      ;; if $len > 0 { continue } else { break }
      (br_if $len_gt_zero (i32.gt_u (local.get $len) (i32.const 0))))

    (i32.const 1)) ;; return 1
)
