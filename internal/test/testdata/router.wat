;; This example module is written in WebAssembly Text Format to show the
;; how a handler works and that it is decoupled from other ABI such as WASI.
;; Most users will prefer a higher-level language such as C, Rust or TinyGo.
(module $router
  ;; get_path writes the request path value to memory, if it isn't larger than
  ;; the buffer size limit. The result is the actual path length in bytes.
  (import "http-handler" "get_path" (func $get_path
    (param $buf i32) (param $buf_limit i32)
    (result (; path_len ;) i32)))

  ;; set_path overwrites the request path with one read from memory.
  (import "http-handler" "set_path" (func $set_path
    (param $path i32) (param $path_len i32)))

  ;; next dispatches control to the next handler on the host.
  (import "http-handler" "next" (func $next))

  ;; set_response_header sets a response header from a name and value read
  ;; from memory
  (import "http-handler" "set_response_header" (func $set_response_header
    (param $name i32) (param $name_len i32)
    (param $value i32) (param $value_len i32)))

  ;; set_response_body overwrites the response body with a value read from memory.
  (import "http-handler" "set_response_body" (func $set_response_body
    (param $body i32)
    (param $body_len i32)))

  ;; http-wasm guests are required to export "memory", so that imported
  ;; functions like "log" can read memory.
  (memory (export "memory") 1 (; 1 page==64KB ;))

  ;; path is an arbitrary area to write data.
  (global $path       i32 (i32.const 1024))
  (global $path_limit i32 (i32.const  256))

  (global $host_prefix i32 (i32.const 0))
  (data (i32.const 0) "/host")
  (global $host_prefix_len i32 (i32.const 5))

  (global $content_type_name i32 (i32.const 32))
  (data (i32.const 32) "Content-Type")
  (global $content_type_name_len i32 (i32.const 12))

  (global $content_type_value i32 (i32.const 64))
  (data (i32.const 64) "text/plain")
  (global $content_type_value_len i32 (i32.const 10))

  (global $body i32 (i32.const 96))
  (data (i32.const 96) "hello world")
  (global $body_len i32 (i32.const 11))

  ;; handle implements a simple HTTP router.
  (func $handle (export "handle")

    (local $path_len i32)

    ;; First, read the path into memory if not larger than our limit.

    ;; path_len = get_path(path, path_limit)
    (local.set $path_len
      (call $get_path (global.get $path) (global.get $path_limit)))

    ;; if path_len > path_limit { next() }
    (if (i32.gt_s (local.get $path_len) (global.get $path_limit))
      (then
        (call $next)
        (return))) ;; dispatch if the path is too long.

    ;; Next, strip any paths starting with '/host' and dispatch.

    ;; if host_prefix_len <= path_len
    (if (i32.eqz (i32.gt_s (global.get $host_prefix_len) (local.get $path_len)))
      (then

        (if (call $memeq ;; path[0:host_prefix_len] == host_prefix
              (global.get $path)
              (global.get $host_prefix)
              (global.get $host_prefix_len))
          (then
            (call $set_path ;; path = path[host_prefix_len:]
              (i32.add (global.get $path)     (global.get $host_prefix_len))
              (i32.sub (local.get  $path_len) (global.get $host_prefix_len)))
            (call $next)
            (return))))) ;; dispatch with the stripped path.

    ;; Otherwise, serve a static response.
    (call $set_response_header
      (global.get $content_type_name)
      (global.get $content_type_name_len)
      (global.get $content_type_value)
      (global.get $content_type_value_len))
    (call $set_response_body
      (global.get $body)
      (global.get $body_len)))

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
