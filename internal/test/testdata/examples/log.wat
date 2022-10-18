;; This example module is written in WebAssembly Text Format to show the
;; how a handler works and that it is decoupled from other ABI such as WASI.
;; Most users will prefer a higher-level language such as C, Rust or TinyGo.
(module $log
  ;; get_headers writes all header values, NUL-terminated, to memory if the
  ;; encoded length isn't larger than `buf-limit`. The result is the encoded
  ;; length in bytes. Ex. "val1\0val2\0" == 10
  (type $get_headers (func
    (param $name i32) (param $name_len i32)
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  ;; get_header_names writes writes all header names, NUL-terminated, to memory
  ;; if the encoded length isn't larger than `$buf_limit. The result is the
  ;; encoded length in bytes. Ex. "Accept\0Date"
  (type $get_header_names (func
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  ;; read_body reads up to $buf_len bytes remaining in the body into memory at
  ;; offset $buf. A zero $buf_len will panic.
  ;;
  ;; The result is `0 or EOF(1) << 32|len`, where `len` is the possibly zero
  ;; length of bytes read.
  (type $read_body (func
    (param $buf i32) (param $buf_len i32)
    (result (; 0 or EOF(1) << 32 | len ;) i64)))

  ;; enable_features tries to enable the given features and returns the entire
  ;; feature bitflag supported by the host.
  (import "http-handler" "enable_features" (func $enable_features
    (param $enable_features i64)
    (result (; enabled_features ;) i64)))

  ;; log logs a message to the host's logs.
  (import "http-handler" "log" (func $log
    (param $message i32) (param $message_len i32)))

  ;; get_method writes the method to memory if it isn't larger than $buf_limit.
  ;; The result is its length in bytes. Ex. "GET"
  (import "http-handler" "get_method" (func $get_method
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  ;; get_uri writes the URI to memory if it isn't larger than $buf_limit.
  ;; The result is its length in bytes. Ex. "/v1.0/hi?name=panda"
  (import "http-handler" "get_uri" (func $get_uri
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  ;; get_protocol_version writes the HTTP protocol version to memory if it
  ;; isn't larger than `buf-limit`. The result is its length in bytes.
  ;; Ex. "HTTP/1.1"
  (import "http-handler" "get_protocol_version" (func $get_protocol_version
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  (import "http-handler" "get_request_header_names" (func $get_request_header_names
    (type $get_header_names)))

  (import "http-handler" "get_request_headers" (func $get_request_headers
    (type $get_headers)))

  (import "http-handler" "read_request_body" (func $read_request_body
    (type $read_body)))

  (import "http-handler" "get_request_trailer_names" (func $get_request_trailer_names
    (type $get_header_names)))

  (import "http-handler" "get_request_trailers" (func $get_request_trailers
    (type $get_headers)))

  ;; next dispatches control to the next handler on the host.
  (import "http-handler" "next" (func $next))

  ;; get_status_code returnts the status code produced by $next.
  (import "http-handler" "get_status_code" (func $get_status_code
    (result (; status_code ;) i32)))

  ;; get_response_header_names requires $feature_buffer_response.
  (import "http-handler" "get_response_header_names" (func $get_response_header_names
    (type $get_header_names)))

  ;; get_response_header requires $feature_buffer_response.
  (import "http-handler" "get_response_headers" (func $get_response_headers
    (type $get_headers)))

  ;; read_response_body requires $feature_buffer_response.
  (import "http-handler" "read_response_body" (func $read_response_body
    (type $read_body)))

  (import "http-handler" "get_response_trailer_names" (func $get_response_trailer_names
    (type $get_header_names)))

  (import "http-handler" "get_response_trailers" (func $get_response_trailers
    (type $get_headers)))

  ;; http-wasm guests are required to export "memory", so that imported
  ;; functions like "log" can read memory.
  (memory (export "memory") 1 1 (; 1 page==64KB ;))
  (global $mem_bytes i32 (i32.const 65536))
  (func $mem_remaining (param $pos i32) (result i32)
    (i32.sub (global.get $mem_bytes) (local.get $pos)))

  ;; define a function table for getting a request or response properties.
  (table 10 funcref)
  (elem (i32.const 0) $get_request_header_names)
  (elem (i32.const 1) $get_request_headers)
  (elem (i32.const 2) $read_request_body)
  (elem (i32.const 3) $get_request_trailer_names)
  (elem (i32.const 4) $get_request_trailers)
  (elem (i32.const 5) $get_response_header_names)
  (elem (i32.const 6) $get_response_headers)
  (elem (i32.const 7) $read_response_body)
  (elem (i32.const 8) $get_response_trailer_names)
  (elem (i32.const 9) $get_response_trailers)
  (func $log_request_headers (call $log_headers (i32.const 0) (i32.const 1)))
  (func $log_request_body (call $log_body (i32.const 2)))
  (func $log_request_trailers (call $log_headers (i32.const 3) (i32.const 4)))
  (func $log_response_headers (call $log_headers (i32.const 5) (i32.const 6)))
  (func $log_response_body (call $log_body (i32.const 7)))
  (func $log_response_trailers (call $log_headers (i32.const 8) (i32.const 9)))

  ;; We don't require the trailers features as it defaults to no-op when
  ;; unsupported.
  ;;
  ;; required_features := feature_buffer_request|feature_buffer_response
  (global $required_features i64 (i64.const 3))

  ;; eof is the upper 32-bits of the $read_body result on EOF.
  (global $eof i64 (i64.const 4294967296)) ;; `1<<32|0`

  ;; must_enable_buffering ensures we can inspect request and response bodies
  ;; without interfering with the next handler.
  (func $must_enable_buffering
    (local $enabled_features i64)

    ;; enabled_features := enable_features(required_features)
    (local.set $enabled_features
      (call $enable_features (global.get $required_features)))

    ;; if enabled_features&required_features == 0 { panic }
    (if (i64.eqz (i64.and
          (local.get $enabled_features)
          (global.get $required_features)))
      (then unreachable)))

  (start $must_enable_buffering)

  ;; handle logs the request and response bodies around the "next" handler.
  (func $handle (export "handle")
    ;; This shows interception before the current request is handled.
    (call $log_request_line)
    (call $log_request_headers)
    (call $log_request_body)
    (call $log_request_trailers)

    ;; This handles the request, in whichever way defined by the host.
    (call $next)

    ;; log("")
    (call $log (i32.const 0) (i32.const 0))

    ;; Because we enabled buffering, we can log the response.
    (call $log_response_line)
    (call $log_response_headers)
    (call $log_response_body)
    (call $log_response_trailers))

  ;; $log_request_line logs the request line. Ex "GET /foo HTTP/1.1"
  (func $log_request_line
    (local $pos i32)
    (local $len i32)

    ;; mem[:len] = method
    (local.set $len
      (call $get_method (local.get $pos) (call $mem_remaining (local.get $pos))))

    ;; pos = len
    (local.set $pos (i32.add (local.get $pos) (local.get $len)))

    ;; mem[pos++] = ' '
    (local.set $pos (call $store_space (local.get $pos)))

    ;; mem[pos:len] = get_uri
    (local.set $len
      (call $get_uri (local.get $pos) (call $mem_remaining (local.get $pos))))

    ;; pos += len
    (local.set $pos (i32.add (local.get $pos) (local.get $len)))

    ;; mem[pos++] = ' '
    (local.set $pos (call $store_space (local.get $pos)))

    ;; mem[pos:len] = get_protocol_version
    (local.set $len
      (call $get_protocol_version (local.get $pos) (call $mem_remaining (local.get $pos))))

    ;; pos += len
    (local.set $pos (i32.add (local.get $pos) (local.get $len)))

    ;; log(mem[:pos])
    (call $log (i32.const 0) (local.get $pos)))

  ;; log_response_line logs the response line, without the status reason.
  ;; Ex. "HTTP/1.1 200"
  (func $log_response_line
    (local $pos i32)
    (local $len i32)
    (local $status_code i32)

    ;; mem[:len] = get_protocol_version
    (local.set $len
      (call $get_protocol_version (local.get $pos) (call $mem_remaining (local.get $pos))))

    ;; pos += len
    (local.set $pos (i32.add (local.get $pos) (local.get $len)))

    ;; mem[pos++] = ' '
    (local.set $pos (call $store_space (local.get $pos)))

    (call $store_status_code (local.get $pos) (call $get_status_code))

    ;; pos += 3
    (local.set $pos (i32.add (local.get $pos) (i32.const 3)))

    ;; log(mem[0:pos])
    (call $log (i32.const 0) (local.get $pos)))

  ;; $log_headers logs all headers in the message.
  (func $log_headers (param $names_fn i32) (param $values_fn i32)
    ;; buf is the current position in memory, initially zero.
    (local $buf i32)

    ;; result is the length of all NUL-terminated values.
    (local $result i32)

    ;; buf_log is where the log function can begin writing.
    (local $buf_log i32)

    ;; name is the position of the current name.
    (local $name i32)

    ;; len is the length of the current NUL-terminated name, exclusive of NUL.
    (local $len i32)

    ;; result = table[names_fn](0, mem_bytes)
    (local.set $result
      (call_indirect (type $get_header_names)
        (local.get  $buf)
        (global.get $mem_bytes)
        (local.get  $names_fn)))

    ;; if there are no headers, return
    (if (i32.eqz (local.get $result)) (then (return)))

    ;; if result > mem_bytes { panic }
    (if (i32.gt_u (local.get $result) (global.get $mem_bytes))
       (then (unreachable))) ;; too big so wasn't written

    ;; buf_log = buf + result
    (local.set $buf_log (i32.add (local.get $buf) (local.get $result)))

    ;; We can start writing memory after the NUL-terminated header names.
    (local.set $buf_log (local.get $result))

    ;; loop while we can read a NUL-terminated name.
    (loop $names
      ;; if mem[buf] == NUL
      (if (i32.eqz (i32.load8_u (local.get $buf)))
        (then ;; reached the end of the name

          ;; name = buf -len
          (local.set $name (i32.sub (local.get $buf) (local.get $len)))

          ;; log_header_fields(name, buf_log, mem_bytes, values_fn)
          (call $log_header_fields
            (local.get  $name) (local.get $len)
            (local.get  $buf_log)
            (global.get $mem_bytes) ;; buf_limit
            (local.get  $values_fn))

          (local.set $buf (i32.add (local.get $buf) (i32.const 1))) ;; buf++
          (local.set $len (i32.const 0))) ;; len = 0
        (else
          (local.set $len (i32.add (local.get $len) (i32.const 1))) ;; len++
          (local.set $buf (i32.add (local.get $buf) (i32.const 1))))) ;; buf++

      (local.set $result (i32.sub (local.get $result) (i32.const 1))) ;; result--

      ;; if result > 0 { continue } else { break }
      (br_if $names (i32.gt_u (local.get $result) (i32.const 0)))))

  ;; log_header_fields logs each header field, using the given function table index.
  (func $log_header_fields
    (param $name i32) (param  $name_len i32)
    (param  $buf i32) (param $buf_limit i32)
    (param   $fn i32)

    ;; result is the length of all NUL-terminated values.
    (local $result i32)

    ;; buf_log is where the log function can begin writing.
    (local $buf_log i32)

    ;; value is the position of the current name.
    (local $value i32)

    ;; len is the length of the current NUL-terminated value, exclusive of NUL.
    (local $len i32)

    ;; result = table[headers_fn](mem[name:name_len], mem[buf:buf_limit])
    (local.set $result (call_indirect (type $get_headers)
      (local.get $name) (local.get $name_len)
      (local.get $buf)  (local.get $buf_limit)
      (local.get $fn)))

    ;; if len == 0 { panic }
    (if (i32.eqz (local.get $result))
       (then (unreachable))) ;; header didn't exist

    ;; if result > buf_limit { panic }
    (if (i32.gt_u (local.get $result) (local.get $buf_limit))
       (then (unreachable))) ;; too big so wasn't written

    ;; buf_log = buf + result
    (local.set $buf_log (i32.add (local.get $buf) (local.get $result)))

    ;; loop while we can read a NUL-terminated value.
    (loop $values
      ;; if mem[buf] == NUL
      (if (i32.eqz (i32.load8_u (local.get $buf)))
        (then ;; reached the end of the value

          ;; value = buf - len
          (local.set $value (i32.sub (local.get $buf) (local.get $len)))

          ;; log_header_field(name, value, buf_log)
          (call $log_header_field
            (local.get $name) (local.get $name_len)
            (local.get $value) (local.get $len)
            (local.get $buf_log))

          (local.set $buf (i32.add (local.get $buf) (i32.const 1))) ;; buf++
          (local.set $len (i32.const 0))) ;; len = 0
        (else
          (local.set $len (i32.add (local.get $len) (i32.const 1))) ;; len++
          (local.set $buf (i32.add (local.get $buf) (i32.const 1))))) ;; buf++

      (local.set $result (i32.sub (local.get $result) (i32.const 1))) ;; result--

      ;; if result > 0 { continue } else { break }
      (br_if $values (i32.gt_u (local.get $result) (i32.const 0)))))

  ;; log_header_field logs a header field formatted like "name: value".
  ;;
  ;; Note: This doesn't enforce a buf_limit as the runtime will panic on OOM.
  (func $log_header_field
    (param  $name i32) (param  $name_len i32)
    (param $value i32) (param $value_len i32)
    (param   $buf i32)

    (local $buf_0 i32)

    ;; buf_0 = buf
    (local.set $buf_0 (local.get $buf))

    ;; copy(mem[buf:], mem[name:name_len])
    (memory.copy
      (local.get $buf)
      (local.get $name)
      (local.get $name_len))

    ;; buf = buf + name_len
    (local.set $buf (i32.add (local.get $buf) (local.get $name_len)))

    ;; mem[buf++] = ':'
    (i32.store8 (local.get $buf) (i32.const (; ':'== ;) 58))
    (local.set $buf (i32.add (local.get $buf) (i32.const 1)))

    ;; mem[buf++] = ' '
    (i32.store8 (local.get $buf) (i32.const (; ' '== ;) 32))
    (local.set $buf (i32.add (local.get $buf) (i32.const 1)))

    ;; copy(mem[buf:], mem[value:value_len])
    (memory.copy
      (local.get $buf)
      (local.get $value)
      (local.get $value_len))

    ;; buf = buf + value_len
    (local.set $buf (i32.add (local.get $buf) (local.get $value_len)))

    ;; log(mem[buf_0:(buf - buf_0)])
    (call $log
      (local.get $buf_0)
      (i32.sub (local.get $buf) (local.get $buf_0))))

  ;; log_body logs the body using the given function table index.
  (func $log_body (param $body_fn i32)
    (local $result i64)
    (local $len i32)

    ;; result = table[body_fn](0, mem_bytes)
    (local.set $result
      (call_indirect (type $read_body)
        (i32.const 0)
        (global.get $mem_bytes)
        (local.get $body_fn)))

    ;; len = uint32(result)
    (local.set $len (i32.wrap_i64 (local.get $result)))

    ;; don't log if there was no body
    (if (i32.eqz (local.get $len)) (then (return)))

    ;; if result & eof != eof { panic }
    (if (i64.ne
          (i64.and (local.get $result) (global.get $eof))
          (global.get $eof))
      (then unreachable)) ;; fail as we couldn't buffer the whole response.

    ;; log("")
    (call $log (i32.const 0) (i32.const 0))
    ;; log(mem[0:len])
    (call $log (i32.const 0) (local.get $len)))

  (func $store_space (param $pos i32) (result i32)
    (i32.store8 (local.get $pos) (i32.const (; ' '== ;) 32))
    (i32.add (local.get $pos) (i32.const 1)))

  (func $store_status_code (param $pos i32) (param $status_code i32)
    (local $rem i32)

    ;; if status_code < 100 || status_code >> 599 { panic }
    (if (i32.or
          (i32.lt_u (local.get $status_code) (i32.const 100))
          (i32.gt_u (local.get $status_code) (i32.const 599)))
       (then (unreachable)))

    ;; write the 3 digits backwards, from right to left.
    (local.set $pos (i32.add (local.get $pos) (i32.const 3))) ;; pos += 3

    (loop $status_code_ne_zero
      ;; rem = status_code % 10
      (local.set $rem (i32.rem_u (local.get $status_code) (i32.const 10)))

      ;; mem[--pos] = rem + '0'
      (local.set $pos (i32.sub (local.get $pos) (i32.const 1)))
      (i32.store8
        (local.get $pos)
        (i32.add(local.get $rem) (i32.const (; '0'== ;) 48)))

      ;; status_code /= 10
      (local.set $status_code (i32.div_u (local.get $status_code) (i32.const 10)))

      ;; if $status_code != 0 { continue } else { break }
      (br_if $status_code_ne_zero (i32.ne (local.get $status_code) (i32.const 0)))))
)
