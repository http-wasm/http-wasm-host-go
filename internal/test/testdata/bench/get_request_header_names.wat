;; $ wat2wasm --debug-names get_request_header_names.wat
(module $get_request_header_names
  (import "http-handler" "get_request_header_names"
    (func $get_request_header_names
      (param $buf i32) (param $buf_limit i32)
      (result (; len ;) i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (func $handle (export "handle")
    (local $buf i32)
    (local $len i32)
    (local $count i32)

    ;; read up to 2KB into memory
    (local.set $len
      (call $get_request_header_names (local.get $buf) (i32.const 2048)))

    ;; if len == 0 { return }
    (if (i32.eqz (local.get $len))
      (then (return))) ;; no headers

    ;; if $len > 2048 { retry }
    (if (i32.gt_u (local.get $len) (i32.const 2048))
       (then
         (call $get_request_header_names (local.get $buf) (local.get $len))
         (drop)))

    ;; loop while we can read a NUL-terminated name.
    (loop $names
      ;; if mem[buf] == NUL
      (if (i32.eqz (i32.load8_u (local.get $buf)))
        (then ;; reached the end of the name
          ;; count++
          (local.set $count (i32.add (local.get $count) (i32.const 1)))))

      (local.set $buf (i32.add (local.get $buf) (i32.const 1))) ;; buf++
      (local.set $len (i32.sub (local.get $len) (i32.const 1))) ;; len--

      ;; if len > 0 { continue } else { break }
      (br_if $names (i32.gt_u (local.get $len) (i32.const 0))))

    ;; if count == 0 { panic }
    (if (i32.eqz (local.get $count))
      (then (unreachable)))) ;; the result wasn't NUL-terminated
)
