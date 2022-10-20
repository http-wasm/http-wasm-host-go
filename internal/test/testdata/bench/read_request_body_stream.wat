;; $ wat2wasm --debug-not_eof read_request_body_stream.wat
(module $read_request_body_stream
  (import "http-handler" "read_request_body" (func $read_request_body
    (param $buf i32) (param $buf_len i32)
    (result (; 0 or EOF(1) << 32 | len ;) i64)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  ;; eof is the upper 32-bits of the $read_request_body result on EOF.
  (global $eof i64 (i64.const 4294967296)) ;; `1<<32|0`

  (func $handle (export "handle")
    (local $result i64)

    (loop $not_eof
      ;; read up to 2KB into memory
      (local.set $result
        (call $read_request_body (i32.const 0) (i32.const 2048)))

      ;; if result & eof != eof { continue } else { break }
      (br_if $not_eof (i64.ne
        (i64.and (local.get $result) (global.get $eof))
        (global.get $eof)))))
)
