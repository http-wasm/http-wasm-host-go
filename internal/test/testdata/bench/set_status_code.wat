;; $ wat2wasm --debug-names set_status_code.wat
(module $set_status_code
  (import "http-handler" "set_status_code"
    (func $set_status_code (param i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (func $handle (export "handle")
    (call $set_status_code
      (i32.const 404))
  )
)
