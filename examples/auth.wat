;; This example module is written in WebAssembly Text Format to show the
;; how a handler works and that it is decoupled from other ABI such as WASI.
;; Most users will prefer a higher-level language such as C, Rust or TinyGo.
(module $auth

  ;; get_header_values writes all header names of the given $kind and $name,
  ;; NUL-terminated, to memory if the encoded length isn't larger than
  ;; $buf_limit. The result is regardless of whether memory was written.
  (import "http_handler" "get_header_values" (func $get_header_values
    (param $kind i32)
    (param $name i32) (param $name_len i32)
    (param $buf i32) (param $buf_limit i32)
    (result (; count << 32| len ;) i64)))

  ;; next dispatches control to the next handler on the host.
  (import "http_handler" "next" (func $next))

  ;; set_header_value overwrites a header of the given $kind and $name with a
  ;; single value.
  (import "http_handler" "set_header_value" (func $set_header_value
    (param $kind i32)
    (param $name i32) (param $name_len i32)
    (param $value i32) (param $value_len i32)))

  ;; set_status_code overrides the status code. The default is 200.
  (import "http_handler" "set_status_code" (func $set_status_code
    (param $status_code i32)))

  ;; http_handler guests are required to export "memory", so that imported
  ;; functions like "get_header" can read memory.
  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  ;; buf is an arbitrary area to write data.
  (global $buf i32 (i32.const 1024))

  (global $authorization_name i32 (i32.const 0))
  (data (i32.const 0) "Authorization")
  (global $authorization_name_len i32 (i32.const 13))

  ;; We expect the username "Aladdin" and password "open sesame".
  (global $authorization_values i32 (i32.const 64))
  (data (i32.const 64) "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==\00")
  (global $authorization_values_len i32 (i32.const 35))

  ;; get_authorization reads the Authorization header into memory
  (func $get_authorization (result (; $count_length ;) i64)
    (call $get_header_values
      (i32.const 0) ;; header_kind_request
      (global.get $authorization_name)
      (global.get $authorization_name_len)
      (global.get $buf)
      (global.get $authorization_values_len)))

  (global $authenticate_name i32 (i32.const 128))
  (data (i32.const 128) "WWW-Authenticate")
  (global $authenticate_name_len i32 (i32.const 16))

  (global $authenticate_value i32 (i32.const 196))
  (data (i32.const 196) "Basic realm=\"test\"")
  (global $authenticate_value_len i32 (i32.const 18))

  ;; set_authenticate adds the WWW-Authenticate header
  (func $set_authenticate
    (call $set_header_value
      (i32.const 1) ;; header_kind_response
      (global.get $authenticate_name)
      (global.get $authenticate_name_len)
      (global.get $authenticate_value)
      (global.get $authenticate_value_len)))

  ;; handle tries BASIC authentication and dispatches to "next" or returns 401.
  (func $handle (export "handle")

    (local $result i64)
    (local $count i32)
    (local $len i32)
    (local $authorization_eq i32)

    (local.set $result (call $get_authorization))

    ;; count = uint32(result >> 32)
    (local.set $count
      (i32.wrap_i64 (i64.shr_u (local.get $result) (i64.const 32))))

    (if (i32.ne (local.get $count) (i32.const 1))
      (then ;; multiple headers, invalid
        (call $set_authenticate)
        (call $set_status_code (i32.const 401))
        (return)))

    ;; len = uint32(result)
    (local.set $len (i32.wrap_i64 (local.get $result)))

    (if (i32.ne (global.get $authorization_values_len) (local.get $len))
      (then ;; authorization_values_length != i32($header_value)
        (call $set_status_code (i32.const 401))
        (return)))

    (local.set $authorization_eq (call $memeq
      (global.get $buf)
      (global.get $authorization_values)
      (global.get $authorization_values_len)))

    (if (i32.eqz (local.get $authorization_eq))
      (then ;; authenticate_value != authorization_values
        (call $set_status_code (i32.const 401)))
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
      (br_if $len_gt_zero (i32.gt_u (local.get $len) (i32.const 0))))

    (i32.const 1)) ;; return 1
)
