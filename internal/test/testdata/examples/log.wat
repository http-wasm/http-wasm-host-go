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

  ;; get_method writes the method to memory if it isn't larger than $buf_limit.
  ;; The result is its length in bytes. Ex. "GET"
  (import "http-handler" "get_method" (func $get_method
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  ;; get_uri writes the URI to memory if it isn't larger than $buf_limit.
  ;; The result is its length in bytes. Ex. "/v1.0/hi?name=panda"
  (import "http-handler" "get_uri" (func $get_uri
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  ;; get_protocol_version writes the HTTP protocol version to memory if it
  ;; isn't larger than `buf-limit`. The result is its length in bytes.
  ;; Ex. "HTTP/1.1"
  (import "http-handler" "get_protocol_version" (func $get_protocol_version
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  ;; get_status_code returnts the status code produced by $next.
  (import "http-handler" "get_status_code" (func $get_status_code
    (result (; status_code ;) i32)))

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
  (memory (export "memory") 1 1 (; 1 page==64KB ;))
  (global $mem_bytes i32 (i32.const 65536))
  (func $mem_remaining (param $pos i32) (result i32)
    (i32.sub (global.get $mem_bytes) (local.get $pos)))

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

  ;; handle logs the request and response bodies around the "next" handler.
  (func $handle (export "handle")
    ;; This shows interception before the current request is handled.
    (call $log_request_line)
    (call $log_body (i32.const 0))

    ;; This handles the request, in whichever way defined by the host.
    (call $next)

    ;; log("")
    (call $log (i32.const 0) (i32.const 0))

    ;; Because we enabled buffering, we can log the response.
    (call $log_response_line)
    (call $log_body (i32.const 1)))

  ;; $log_request_line logs the request line. Ex "GET /foo HTTP/1.1"
  (func $log_request_line
    (local $pos i32)
    (local $len i32)

    ;; mem[:len] = method
    (local.set $len
      (call $get_method (local.get $pos) (call $mem_remaining (local.get $pos))))

    ;; pos = len
    (local.set $pos (i32.add (local.get $pos) (local.get $len)))

    ;; mem[pos++] = ' '
    (local.set $pos (call $store_space (local.get $pos)))

    ;; mem[pos:len] = get_uri
    (local.set $len
      (call $get_uri (local.get $pos) (call $mem_remaining (local.get $pos))))

    ;; pos += len
    (local.set $pos (i32.add (local.get $pos) (local.get $len)))

    ;; mem[pos++] = ' '
    (local.set $pos (call $store_space (local.get $pos)))

    ;; mem[pos:len] = get_protocol_version
    (local.set $len
      (call $get_protocol_version (local.get $pos) (call $mem_remaining (local.get $pos))))

    ;; pos += len
    (local.set $pos (i32.add (local.get $pos) (local.get $len)))

    ;; log(mem[:pos])
    (call $log (i32.const 0) (local.get $pos)))

  ;; log_response_line logs the response line, without the status reason.
  ;; Ex. "HTTP/1.1 200"
  (func $log_response_line
    (local $pos i32)
    (local $len i32)
    (local $status_code i32)

    ;; mem[:len] = get_protocol_version
    (local.set $len
      (call $get_protocol_version (local.get $pos) (call $mem_remaining (local.get $pos))))

    ;; pos += len
    (local.set $pos (i32.add (local.get $pos) (local.get $len)))

    ;; mem[pos++] = ' '
    (local.set $pos (call $store_space (local.get $pos)))

    (call $store_status_code (local.get $pos) (call $get_status_code))

    ;; pos += 3
    (local.set $pos (i32.add (local.get $pos) (i32.const 3)))

    ;; log(mem[0:pos])
    (call $log (i32.const 0) (local.get $pos)))

  ;; log_body logs the body using the given function table index.
  (func $log_body (param $body_fn i32)
    (local $len i32)

    ;; len = table[body_fn](0, mem_bytes)
    (local.set $len
      (call_indirect (type $get_body)
        (i32.const 0)
        (global.get $mem_bytes)
        (local.get $body_fn)))

    (if (i32.eqz (local.get $len)) (then (return)))

    ;; log("")
    (call $log (i32.const 0) (i32.const 0))
    ;; log(mem[0:len])
    (call $log (i32.const 0) (local.get $len)))

  (func $store_space (param $pos i32) (result i32)
    (i32.store8 (local.get $pos) (i32.const (; ' '== ;) 32))
    (i32.add (local.get $pos) (i32.const 1)))

  (func $store_status_code (param $pos i32) (param $status_code i32)
    (local $rem i32)

    ;; if status_code < 100 || status_code >> 599 { panic }
    (if (i32.or
          (i32.lt_u (local.get $status_code) (i32.const 100))
          (i32.gt_u (local.get $status_code) (i32.const 599)))
       (then (unreachable)))

    ;; We will write the 3 digits backwards, from right to left.
    (local.set $pos (i32.add (local.get $pos) (i32.const 3))) ;; pos += 3

    (loop $status_code_ne_zero
      ;; rem = status_code % 10
      (local.set $rem (i32.rem_u (local.get $status_code) (i32.const 10)))

      ;; mem[--pos] = rem + '0'
      (local.set $pos (i32.sub (local.get $pos) (i32.const 1)))
      (i32.store8
        (local.get $pos)
        (i32.add(local.get $rem) (i32.const (; '0'== ;) 48)))

      ;; status_code /= 10
      (local.set $status_code (i32.div_u (local.get $status_code) (i32.const 10)))

      ;; if $status_code != 0 { continue } else { break }
      (br_if $status_code_ne_zero (i32.ne (local.get $status_code) (i32.const 0)))))
)
