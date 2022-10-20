(module $protocol_version

  (import "http-handler" "get_protocol_version" (func $get_protocol_version
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  (import "http-handler" "write_body" (func $write_body
    (param $kind i32)
    (param $buf i32) (param $buf_len i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  ;; handle writes the protocol version to the response body.
  (func (export "handle")
    (local $len i32)

    ;; read the protocol version into memory at offset zero.
    (local.set $len
      (call $get_protocol_version (i32.const 0) (i32.const 1024)))

    ;; write the protocol version to the response body.
    (call $write_body
      (i32.const 1) ;; body_kind_response
      (i32.const 0) (local.get $len)))
)
