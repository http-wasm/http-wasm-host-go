package mosn_test

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	requestBody  = "{\"hello\": \"panda\"}"
	responseBody = "{\"hello\": \"world\"}"
)

func Example_auth() {
	config := filepath.Join("testdata", "config-auth.json")
	startMosn(config)
	defer stopMosn(config)

	resp, err := http.Get("http://localhost:2046")
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()

	// Invoke some requests, only one of which should pass
	headers := []http.Header{
		{"NotAuthorization": {"1"}},
		{"Authorization": {""}},
		{"Authorization": {"Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ=="}},
		{"Authorization": {"0"}},
	}

	for _, header := range headers {
		req, err := http.NewRequest(http.MethodGet, "http://localhost:2046", nil)
		if err != nil {
			log.Panicln(err)
		}
		req.Header = header

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Panicln(err)
		}
		resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			fmt.Println("OK")
		case http.StatusUnauthorized:
			fmt.Println("Unauthorized")
		default:
			log.Panicln("unexpected status code", resp.StatusCode)
		}
	}

	// Output:
	// Unauthorized
	// Unauthorized
	// OK
	// Unauthorized
}

func Example_log() {
	config := filepath.Join("testdata", "config-log.json")
	startMosn(config)
	defer stopMosn(config)

	resp, err := http.Post("http://localhost:2046", "application/json", strings.NewReader(requestBody))
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()

	// Ensure the response body was still readable!
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panicln(err)
	}
	if want, have := responseBody, string(body); want != have {
		log.Panicf("unexpected response body, want: %q, have: %q", want, have)
	}

	// Output:
	// request body:
	// {"hello": "panda"}
	// response body:
	// {"hello": "world"}
}

func startMosn(configPath string) {
	cmd := exec.Command("./mosntest", "start", "--config", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Panicln(err)
	}
	for i := 0; i < 100; i++ {
		time.Sleep(200 * time.Millisecond)
		resp, err := http.Get("http://localhost:34901")
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return
		}
	}
	log.Panicln("start mosn failed")
}

func stopMosn(configPath string) {
	cmd := exec.Command("./mosntest", "stop", "--config", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Panicln(err)
	}
}

func init() {
	goCmd := filepath.Join(runtime.GOROOT(), "bin", "go")
	cmd := exec.Command(goCmd, "build", "-o", "mosntest", "./internal/cmd")
	if err := cmd.Run(); err != nil {
		log.Panicln(err)
	}
}
