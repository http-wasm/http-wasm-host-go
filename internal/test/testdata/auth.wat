;; This example module is written in WebAssembly Text Format to show the
;; how a handler works and that it is decoupled from other ABI such as WASI.
;; Most users will prefer a higher-level language such as C, Rust or TinyGo.
(module $auth

  ;; read_request_header writes a header value to memory if it exists and isn't
  ;; larger than the buffer size limit. The result is`1<<32|value_len` or zero
  ;; if the header doesn't exist.
  (import "http-handler" "read_request_header"
    (func $read_request_header
      (param $name i32) (param $name_len i32)
      (param $buf i32) (param $value_limit i32)
      (result (; 0 or 1 << 32| value_len ;) i64)))

  ;; next instructs the host to invoke the next handler.
  (import "http-handler" "next" (func $next))

  ;; set_response_header sets a response header.
  (import "http-handler" "set_response_header"
    (func $set_response_header
      (param $name i32) (param $name_len i32)
      (param $value i32) (param $value_len i32)))

  ;; next instructs the host to invoke the next handler.
  (import "http-handler" "send_response"
    (func $send_response
      (param $status_code i32)
      (param $body i32)
      (param $body_len i32)))

  ;; http-wasm guests are required to export "memory", so that imported
  ;; functions like "read_request_header" can read memory.
  (memory (export "memory") 1 (; 1 page==64KB ;))

  ;; buf is an arbitrary area to write data
  (global $buf i32 (i32.const 1024))

  (global $authorization_name i32 (i32.const 0))
  (data (i32.const 0) "Authorization")
  (global $authorization_name_len i32 (i32.const 13))

  ;; We expect the username "Aladdin" and password "open sesame".
  (global $authorization_value i32 (i32.const 32))
  (data (i32.const 32) "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==")
  (global $authorization_value_len i32 (i32.const 35))

  ;; get_authorization reads the Authorization header into memory
  (func $get_authorization (result (; $exists_length ;) i64)
    (call $read_request_header
      (global.get $authorization_name)
      (global.get $authorization_name_len)
      (global.get $buf)
      (global.get $authorization_value_len)))

  (global $authenticate_name i32 (i32.const 64))
  (data (i32.const 64) "WWW-Authenticate")
  (global $authenticate_name_len i32 (i32.const 16))

  (global $authenticate_value i32 (i32.const 96))
  (data (i32.const 96) "Basic realm=\"test\"")
  (global $authenticate_value_len i32 (i32.const 35))

  ;; set_authenticate adds the WWW-Authenticate header
  (func $set_authenticate
    (call $set_response_header
      (global.get $authenticate_name)
      (global.get $authenticate_name_len)
      (global.get $authenticate_value)
      (global.get $authenticate_value_len)))

  ;; handle tries BASIC authentication and dispatches to "next" or returns 401.
  (func $handle (export "handle")

    (local $header_value i64) ;; == $ok|value_length
    (call $get_authorization) ;; stack: [$header_value]
    (local.set $header_value)

    (if (i64.eqz (local.get $header_value))
      (then ;; authorization required, but no header
        (call $set_authenticate)
        (call $send_response (i32.const 401) (i32.const 0) (i32.const 0))
        (return)))

    (if (i32.eq (global.get $authorization_value_len) (i32.wrap_i64 (local.get $header_value)))
      (then ;; authorization_value_length != i32($header_value)
        (call $send_response (i32.const 401) (i32.const 0) (i32.const 0))
        (return)))

    (call $memeq
      (global.get $buf)
      (global.get $authorization_value)
      (global.get $authorization_value_len))

    (if (i32.eqz)
      (then ;; authenticate_value != authorization_value
        (call $send_response (i32.const 401) (i32.const 0) (i32.const 0)))
      (else ;; authorization passed! call the next handler
        (call $next))))

  ;; memeq is like memcmp except it returns 0 (ne) or 1 (eq)
  (func $memeq (param $ptr1 i32) (param $ptr2 i32) (param $len i32) (result i32)
    (local $i1 i32)
    (local $i2 i32)
    (local.set $i1 (local.get $ptr1)) ;; i1 := ptr1
    (local.set $i2 (local.get $ptr2)) ;; i2 := ptr1

    (loop
      ;; if mem[i1] != mem[i2]
      (if (i32.ne (i32.load8_u (local.get $i1)) (i32.load8_u (local.get $i2)))
        (then (return (i32.const 0)))) ;; return 0

      (local.set $i1 (i32.add (local.get $i1) (i32.const 1))) ;; i1++
      (local.set $i2 (i32.add (local.get $i2) (i32.const 1))) ;; i2++
      (local.set $len (i32.sub (local.get $len (i32.const 1)))) ;; $num--

      ;; if $len > 0 { continue } else { break }
      (br_if 0 (i32.gt_s (local.get $len) (i32.const 0))))

    (i32.const 1)) ;; return 1
)
