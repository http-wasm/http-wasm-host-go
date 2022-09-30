;; This example module is written in WebAssembly Text Format to show the
;; how a handler works and that it is decoupled from other ABI such as WASI.
;; Most users will prefer a higher-level language such as C, Rust or TinyGo.
(module $log
    ;; http.log writes a message to the host console.
    (import "http" "log" (func $http.log (param $ptr i32) (param $size i32)))

    ;; http.next instructs the host to invoke the next handler.
    (import "http" "next" (func $http.next))

    ;; http-wasm guests are required to export "memory", so that host functions
    ;; such as http.log can read memory.
    (memory (export "memory") 1 (; 1 page==64KB ;))

    ;; load constants into memory used for http.log.
    (data (i32.const 0) "before")
    (data (i32.const 8) "after")

    ;; http-wasm guests are required to export "handle", invoked per request.
    (func $handle (export "handle")
        ;; This shows interception before the current request is handled.
        (call $http.log
            (i32.const 0) ;; offset of "before"
            (i32.const 6) ;; size of "before"
        )

        ;; This handles the request, in whichever way defined by the host.
        (call $http.next)

        ;; This shows interception after the current request is handled.
        (call $http.log
            (i32.const 8) ;; offset of "after"
            (i32.const 5) ;; size of "after"
        )
    )
)
