(module $log

  (import "http_handler" "log" (func $log
    (param $level i32)
    (param $buf i32) (param $buf_limit i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))
  (global $message i32 (i32.const 0))
  (data (i32.const 0) "hello world")
  (global $message_len i32 (i32.const 11))

  (func $handle (export "handle")
    (call $log
      (i32.const 1) ;; INFO
      (global.get $message)
      (global.get $message_len)))
)
