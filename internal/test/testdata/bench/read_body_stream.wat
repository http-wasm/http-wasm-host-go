(module $read_body_stream

  (import "http_handler" "read_body" (func $read_body
    (param $kind i32)
    (param $buf i32) (param $buf_len i32)
    (result (; 0 or EOF(1) << 32 | len ;) i64)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  ;; eof is the upper 32-bits of the $read_body result on EOF.
  (global $eof i64 (i64.const 4294967296)) ;; `1<<32|0`

  (func (export "handle_request") (result (; ctx_next ;) i64)
    (local $result i64)

    (loop $not_eof
      ;; read up to 2KB into memory
      (local.set $result
        (call $read_body
          (i32.const 0) ;; body_kind_request
          (i32.const 0) (i32.const 2048)))

      ;; if result & eof != eof { continue } else { break }
      (br_if $not_eof (i64.ne
        (i64.and (local.get $result) (global.get $eof))
        (global.get $eof))))

    ;; skip any next handler as the benchmark is about read_body.
    (return (i64.const 0)))

  ;; handle_response should not be called as handle_request returns zero.
  (func (export "handle_response") (param $reqCtx i32) (param $is_error i32)
    (unreachable))
)
