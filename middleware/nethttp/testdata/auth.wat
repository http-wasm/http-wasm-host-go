;; This example module is written in WebAssembly Text Format to show the
;; how a handler works and that it is decoupled from other ABI such as WASI.
;; Most users will prefer a higher-level language such as C, Rust or TinyGo.
(module $auth
    ;; http.read_request_header writes the first header value to memory and
    ;; returns the amount of bytes written.
    (import "http" "read_request_header"
      (func $http.read_request_header
          (param $name_ptr i32) (param $name_size i32)
          (param $value_ptr i32) (param $value_limit i32)
          (result (; value_bytes_written ;) i32)))

    ;; http.next instructs the host to invoke the next handler.
    (import "http" "set_status_code"
      (func $http.set_status_code (param $status_code i32)))

    ;; http.next instructs the host to invoke the next handler.
    (import "http" "next" (func $http.next))

    ;; http-wasm guests are required to export "memory", so that host functions
    ;; such as http.log can read memory.
    (memory (export "memory") 1 (; 1 page==64KB ;))

    ;; load the auth header size.
    (data (i32.const 0) "Authorization")

    ;; http-wasm guests are required to export "handle", invoked per request.
    (func $handle (export "handle")
        ;; First, read the Authorization header.
        (call $http.read_request_header
                   (i32.const  0) ;; offset of "Authorization"
                   (i32.const 13) ;; size of "Authorization"
                   (i32.const 16) ;; where to write the value
                   (i32.const  2) ;; max bytes to write.
        )                         ;; stack: [$value_length]

        ;; The valid auth header is ASCII '1' which is 49 in decimal.
        ;; If we add the length written to the value, we should get 50.
        (i32.load (i32.const 16)) ;; stack: [$value_length,  $value[0]]
        (i32.add)                 ;; stack: [$value_length + $value[0]]
        (i32.const 50)            ;; stack: [$value_length + $value[0], 50]
        (i32.eq)

        (if
          (then ;; auth passed! call the next handler
             (call $http.next))
          (else ;; auth failed!
             (call $http.set_status_code (i32.const 401)))
        )
    )
)
