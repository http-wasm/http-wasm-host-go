;; This example module is written in WebAssembly Text Format to show the
;; how a handler works and that it is decoupled from other ABI such as WASI.
;; Most users will prefer a higher-level language such as C, Rust or TinyGo.
(module $auth

  ;; get_request_header writes a header value to memory if it exists and isn't
  ;; larger than the buffer size limit. The result is `1<<32|value_len` or zero
  ;; if the header doesn't exist. `value_len` is the actual value length in
  ;; bytes.
  (import "http-handler" "get_request_header" (func $get_request_header
    (param $name i32) (param $name_len i32)
    (param $value i32) (param $value_limit i32)
    (result (; 0 or 1 << 32| value_len ;) i64)))

  ;; next dispatches control to the next handler on the host.
  (import "http-handler" "next" (func $next))

  ;; set_response_header sets a response header from a name and value read
  ;; from memory
  (import "http-handler" "set_response_header" (func $set_response_header
    (param $name i32) (param $name_len i32)
    (param $value i32) (param $value_len i32)))

  ;; next instructs the host to invoke the next handler.
  (import "http-handler" "send_response" (func $send_response
    (param $status_code i32)
    (param $body i32)
    (param $body_len i32)))

  ;; http-wasm guests are required to export "memory", so that imported
  ;; functions like "get_request_header" can read memory.
  (memory (export "memory") 1 (; 1 page==64KB ;))

  ;; buf is an arbitrary area to write data.
  (global $buf i32 (i32.const 1024))

  (global $authorization_name i32 (i32.const 0))
  (data (i32.const 0) "Authorization")
  (global $authorization_name_len i32 (i32.const 13))

  ;; We expect the username "Aladdin" and password "open sesame".
  (global $authorization_value i32 (i32.const 64))
  (data (i32.const 64) "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==")
  (global $authorization_value_len i32 (i32.const 34))

  ;; get_authorization reads the Authorization header into memory
  (func $get_authorization (result (; $exists_length ;) i64)
    (call $get_request_header
      (global.get $authorization_name)
      (global.get $authorization_name_len)
      (global.get $buf)
      (global.get $authorization_value_len)))

  ;; clear_buf clears any memory that may have been written.
  (func $clear_buf (param $len i32)
    (memory.fill
      (global.get $buf)
      (local.get  $len)
      (i32.const  0)))

  (global $authenticate_name i32 (i32.const 128))
  (data (i32.const 128) "WWW-Authenticate")
  (global $authenticate_name_len i32 (i32.const 16))

  (global $authenticate_value i32 (i32.const 196))
  (data (i32.const 196) "Basic realm=\"test\"")
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

    (local $authorization_len i32)
    (local $authorization_eq i32)

    (local.set $authorization_len
      (i32.wrap_i64 (call $get_authorization)))

    (if (i32.eqz (local.get $authorization_len))
      (then ;; authorization required, but no header
        (call $clear_buf (local.get $authorization_len))
        (call $set_authenticate)
        (call $send_response (i32.const 401) (i32.const 0) (i32.const 0))
        (return)))

    (if (i32.ne (global.get $authorization_value_len) (local.get $authorization_len))
      (then ;; authorization_value_length != i32($header_value)
        (call $clear_buf (local.get $authorization_len))
        (call $send_response (i32.const 401) (i32.const 0) (i32.const 0))
        (return)))

    (local.set $authorization_eq (call $memeq
      (global.get $buf)
      (global.get $authorization_value)
      (global.get $authorization_value_len)))

    (call $clear_buf (local.get $authorization_len))

    (if (i32.eqz (local.get $authorization_eq))
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

    (loop $len_gt_zero
      ;; if mem[i1] != mem[i2]
      (if (i32.ne (i32.load8_u (local.get $i1)) (i32.load8_u (local.get $i2)))
        (then (return (i32.const 0)))) ;; return 0

      (local.set $i1  (i32.add (local.get $i1)  (i32.const 1))) ;; i1++
      (local.set $i2  (i32.add (local.get $i2)  (i32.const 1))) ;; i2++
      (local.set $len (i32.sub (local.get $len) (i32.const 1))) ;; $len--

      ;; if $len > 0 { continue } else { break }
      (br_if $len_gt_zero (i32.gt_s (local.get $len) (i32.const 0))))

    (i32.const 1)) ;; return 1
)
