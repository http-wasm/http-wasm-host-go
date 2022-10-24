(module $get_header_names

  (import "http_handler" "get_header_names" (func $get_header_names
    (param $kind i32)
    (param $buf i32) (param $buf_limit i32)
    (result (; count << 32| len ;) i64)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  (func $handle (export "handle")
    (local $buf i32)
    (local $result i64)
    (local $len i32)
    (local $expected_count i32)
    (local $count i32)

    ;; read up to 2KB into memory
    (local.set $result
      (call $get_header_names
        (i32.const 0) ;; header_kind_request
        (local.get $buf) (i32.const 2048)))

    ;; if result == 0 { return }
    (if (i64.eqz (local.get $result))
      (then (return))) ;; no headers

    ;; expected_count = uint32(result >> 32)
    (local.set $expected_count
      (i32.wrap_i64 (i64.shr_u (local.get $result) (i64.const 32))))

    ;; len = uint32(result)
    (local.set $len (i32.wrap_i64 (local.get $result)))

    ;; if $len > 2048 { retry }
    (if (i32.gt_u (local.get $len) (i32.const 2048))
       (then
         (drop (call $get_header_names
           (i32.const 0) ;; header_kind_request
           (local.get $buf) (local.get $len)))))

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

    ;; if count != expected_count { panic }
    (if (i32.eq (local.get $count) (local.get $expected_count))
      (then (unreachable)))) ;; the result wasn't NUL-terminated
)
