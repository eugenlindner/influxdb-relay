package relay

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/vente-privee/influxdb-relay/config"
)

var (
	ti         = time.Now()
	promBody   = Body{}
	influxBody = Body{}
	adminBody  = Body{}

	basicPingWriter = &ResponseWriter{
		writeBuf: &bytes.Buffer{},
		header: http.Header{
			"X-Influxdb-Version": []string{"relay"},
		},
		code: 204,
	}
	teapotPingWriter = &ResponseWriter{
		writeBuf: &bytes.Buffer{},
		header: http.Header{
			"X-Influxdb-Version": []string{"relay"},
			"Content-Length":     []string{"0"},
		},
		code: http.StatusTeapot,
	}
	wrongMethodPingWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"" + http.StatusText(http.StatusMethodNotAllowed) + "\"")),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{fmt.Sprintf("%d", len(http.StatusText(http.StatusMethodNotAllowed))+2)},
		},
		code: http.StatusMethodNotAllowed,
	}
	basicStatusWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"\\\"status\\\": {\\\"test\\\":{\\\"location\\\":\\\"\\\"}}\"")),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"44"},
		},
		code: http.StatusOK,
	}
	wrongMethodStatusWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"" + http.StatusText(http.StatusMethodNotAllowed) + "\"")),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{fmt.Sprintf("%d", len(http.StatusText(http.StatusMethodNotAllowed))+2)},
		},
		code: http.StatusMethodNotAllowed,
	}
	wrongMethodPromWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"" + http.StatusText(http.StatusMethodNotAllowed) + "\"")),
		header: http.Header{
			"Allow":          []string{http.MethodPost},
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{fmt.Sprintf("%d", len(http.StatusText(http.StatusMethodNotAllowed))+2)},
		},
		code: http.StatusMethodNotAllowed,
	}
	wrongBackendPromWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"unable to write points\"")),
		header: http.Header{
			"Allow":          []string{http.MethodPost},
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"24"},
		},
		code: http.StatusServiceUnavailable,
	}
	BackendDownPromWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"unable to write points\"")),
		header: http.Header{
			"Allow":          []string{http.MethodPost},
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"24"},
		},
		code: http.StatusServiceUnavailable,
	}
	BackendUpPromWriter = &ResponseWriter{
		writeBuf: &bytes.Buffer{},
		header: http.Header{
			"Content-Type": []string{"text/plain"},
		},
		code: http.StatusNoContent,
	}
	BackendUpPromError400Writer = &ResponseWriter{
		writeBuf: &bytes.Buffer{},
		header: http.Header{
			"Content-Length": []string{"0"},
			"Content-Type":   []string{"text/plain"},
		},
		code: http.StatusBadRequest,
	}
	BackendUpPromError500Writer = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"unable to write points\"")),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"24"},
		},
		code: http.StatusServiceUnavailable,
	}
	wrongMethodInfluxWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"" + http.StatusText(http.StatusMethodNotAllowed) + "\"")),
		header: http.Header{
			"Allow":          []string{http.MethodPost},
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{fmt.Sprintf("%d", len(http.StatusText(http.StatusMethodNotAllowed))+2)},
		},
		code: http.StatusMethodNotAllowed,
	}
	wrongBackendInfluxWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"unable to write points\"")),
		header: http.Header{
			"Allow":          []string{http.MethodPost},
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"24"},
		},
		code: http.StatusServiceUnavailable,
	}
	BackendDownInfluxWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"unable to write points\"")),
		header: http.Header{
			"Allow":          []string{http.MethodPost},
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"24"},
		},
		code: http.StatusServiceUnavailable,
	}
	BackendUpInfluxWriter = &ResponseWriter{
		writeBuf: &bytes.Buffer{},
		header: http.Header{
			"Content-Type": []string{"text/plain"},
		},
		code: http.StatusNoContent,
	}
	BackendUpInfluxError400Writer = &ResponseWriter{
		writeBuf: &bytes.Buffer{},
		header: http.Header{
			"Content-Length": []string{"0"},
			"Content-Type":   []string{"text/plain"},
		},
		code: http.StatusBadRequest,
	}
	BackendUpInfluxError500Writer = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"unable to write points\"")),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"24"},
		},
		code: http.StatusServiceUnavailable,
	}
	InfluxFailParsePointWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"unable to parse points\"")),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"24"},
		},
		code: http.StatusBadRequest,
	}
	InfluxParsePointWriter = &ResponseWriter{
		writeBuf: &bytes.Buffer{},
		header: http.Header{
			"Content-Type": []string{"text/plain"},
		},
		code: http.StatusNoContent,
	}
	AdminWriter = &ResponseWriter{
		writeBuf: &bytes.Buffer{},
		header:   http.Header{},
		code:     http.StatusNoContent,
	}
	AdminWriterNoBackEnds = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte(`"unable to forward query"`)),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"25"},
		},
		code: http.StatusServiceUnavailable,
	}
	AdminWriterNoAdminBackEnds = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte(`"unable to forward query"`)),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"25"},
		},
		code: http.StatusServiceUnavailable,
	}
	AdminWriterWrongMethod = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"" + http.StatusText(http.StatusMethodNotAllowed) + "\"")),
		header: http.Header{
			"Allow":          []string{http.MethodPost},
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{fmt.Sprintf("%d", len(http.StatusText(http.StatusMethodNotAllowed))+2)},
		},
		code: http.StatusMethodNotAllowed,
	}
	AdminWriterServerError = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte(`"unable to forward query"`)),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"25"},
		},
		code: http.StatusServiceUnavailable,
	}
	AdminWriterClientError = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\n")),
		header:   http.Header{},
		code:     http.StatusServiceUnavailable,
	}
	AdminWriterWrongURL = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte(`"unable to forward query"`)),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"25"},
		},
		code: http.StatusServiceUnavailable,
	}
	AdminWriterUnreachableURL = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte(`"unable to forward query"`)),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"25"},
		},
		code: http.StatusServiceUnavailable,
	}
)

var (
	emptyConfig      = config.HTTPConfig{}
	configPingTeapot = config.HTTPConfig{
		DefaultPingResponse: http.StatusTeapot,
	}
)

func captureOutput(f func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	f()
	log.SetOutput(os.Stderr)
	return buf.String()
}

var (
	Error500    *httptest.Server
	Error400    *httptest.Server
	ValidServer *httptest.Server
)

func TestMain(m *testing.M) {
	Error500 = httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusInternalServerError)
	}))
	Error400 = httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusBadRequest)
	}))
	ValidServer = httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
	}))
	defer Error500.Close()
	defer Error400.Close()
	defer ValidServer.Close()

	m.Run()
}

func TestHandlePingSimple(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)
	r, err := http.NewRequest("GET", "influxdb", emptyBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handlePing(w, r, ti)
	WriterTest(t, basicPingWriter, w)
}

func TestHandlePingTeapot(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, configPingTeapot, false)
	r, err := http.NewRequest("GET", "influxdb", emptyBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handlePing(w, r, ti)
	WriterTest(t, teapotPingWriter, w)
}

func TestHandlePingWrongMethod(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, configPingTeapot, false)
	r, err := http.NewRequest("OPTIONS", "influxdb", emptyBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handlePing(w, r, ti)
	WriterTest(t, wrongMethodPingWriter, w)
}

func TestHandleStatusSimple(t *testing.T) {
	defer resetWriter()
	cfgOut := config.HTTPOutputConfig{Name: "test", InputType: "influxdb"}
	h := createHTTP(t, emptyConfig, false)
	r, err := http.NewRequest("GET", "influxdb", emptyBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOut)
	h.backends = append(h.backends, b)
	h.handleStatus(w, r, ti)
	WriterTest(t, basicStatusWriter, w)
	h.backends = h.backends[:0]
}

func TestHandleStatusWrongMethod(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)
	r, err := http.NewRequest("OPTIONS", "influxdb", emptyBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handleStatus(w, r, ti)
	WriterTest(t, wrongMethodStatusWriter, w)
}

func TestHandlePromWrongMethod(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)
	r, err := http.NewRequest("WRONG", "influxdb", emptyBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handleProm(w, r, ti)
	WriterTest(t, wrongMethodPromWriter, w)
}

func TestHandlePromWrongBackend(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)
	promBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest("OPTIONS", "influxdb", promBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handleProm(w, r, ti)
	WriterTest(t, wrongBackendPromWriter, w)
}

func TestHandlePromBackendDown(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)
	cfgOutProm := config.HTTPOutputConfig{Name: "test_prometheus", InputType: "prometheus"}
	cfgOutInflux := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb"}
	promBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest("OPTIONS", "influxdb", promBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutInflux)
	b2, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.backends = append(h.backends, b2)
	output := captureOutput(func() {
		h.handleProm(w, r, ti)
	})
	output = output[20:]
	assert.Equal(t, "Problem posting to relay \"http://\" backend \"test_prometheus\": Post : unsupported protocol scheme \"\"\n", output)
	WriterTest(t, BackendDownPromWriter, w)
	h.backends = h.backends[:0]
}

func TestHandlePromBackendUp(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)

	cfgOutProm := config.HTTPOutputConfig{Name: "test_prometheus", InputType: "prometheus", Location: ValidServer.URL + "/prom"}
	promBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest(http.MethodPost, ValidServer.URL, promBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.handleProm(w, r, ti)
	WriterTest(t, BackendUpPromWriter, w)
	h.backends = h.backends[:0]
}

func TestHandlePromBackendUpError400(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)

	cfgOutProm := config.HTTPOutputConfig{Name: "test_prometheus", InputType: "prometheus", Location: Error400.URL + "/prom"}
	promBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest(http.MethodPost, Error400.URL, promBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.handleProm(w, r, ti)
	WriterTest(t, BackendUpPromError400Writer, w)
	h.backends = h.backends[:0]
}

func TestHandlePromBackendUpError500(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)
	cfgOutProm := config.HTTPOutputConfig{Name: "test_prometheus", InputType: "prometheus", Location: Error500.URL + "/prom"}
	promBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest(http.MethodPost, Error500.URL, promBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	output := captureOutput(func() {
		h.handleProm(w, r, ti)
	})
	output = output[20:]
	assert.Equal(t, `5xx response for relay "http://" backend "test_prometheus": 500
`, output)
	WriterTest(t, BackendUpPromError500Writer, w)
	h.backends = h.backends[:0]
}

func TestHandleInfluxWrongMethod(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)
	r, err := http.NewRequest("TESTING", "influxdb", emptyBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handleStandard(w, r, ti)
	WriterTest(t, wrongMethodInfluxWriter, w)
}

func TestHandleInfluxWrongBackend(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)
	influxBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest(http.MethodOptions, "influxdb", influxBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handleStandard(w, r, ti)
	WriterTest(t, wrongBackendInfluxWriter, w)
}

func TestHandleInfluxBackendDown(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)
	cfgOutProm := config.HTTPOutputConfig{Name: "test_prometheus", InputType: "prometheus"}
	cfgOutInflux := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb"}
	influxBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest(http.MethodOptions, "influxdb", influxBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutInflux)
	b2, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.backends = append(h.backends, b2)
	output := captureOutput(func() {
		h.handleStandard(w, r, ti)
	})
	output = output[20:]
	assert.Equal(t, `Problem posting to relay "http://" backend "test_influx": Post : unsupported protocol scheme ""
`, output)
	WriterTest(t, BackendDownInfluxWriter, w)
	h.backends = h.backends[:0]
}

func TestHandleInfluxBackendUp(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)
	cfgOutProm := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: ValidServer.URL + "/influxdb"}
	influxBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest(http.MethodPost, ValidServer.URL, influxBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.handleStandard(w, r, ti)
	WriterTest(t, BackendUpInfluxWriter, w)
	h.backends = h.backends[:0]
}

func TestHandleInfluxBackendUpError400(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)

	cfgOutProm := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: Error400.URL + "/influxdb"}
	influxBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest(http.MethodPost, Error400.URL, influxBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.handleStandard(w, r, ti)
	WriterTest(t, BackendUpInfluxError400Writer, w)
	h.backends = h.backends[:0]
}

func TestHandleInfluxBackendUpError500(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)

	cfgOutProm := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: Error500.URL + "/influxdb"}
	influxBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest(http.MethodPost, Error500.URL, influxBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	output := captureOutput(func() {
		h.handleStandard(w, r, ti)
	})
	output = output[20:]
	assert.Equal(t, `5xx response for relay "http://" backend "test_influx": 500
`, output)
	WriterTest(t, BackendUpInfluxError500Writer, w)
	h.backends = h.backends[:0]
}

func TestHandleInfluxFailParsePoint(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)

	cfgOutProm := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: ValidServer.URL + "/influxdb"}
	influxBody.buf = bytes.NewBuffer([]byte("Some Bug"))
	r, err := http.NewRequest(http.MethodPost, ValidServer.URL, influxBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.handleStandard(w, r, ti)
	WriterTest(t, InfluxFailParsePointWriter, w)
	h.backends = h.backends[:0]
}

func TestHandleInfluxParsePoint(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)

	cfgOutProm := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: ValidServer.URL + "/influxdb"}
	influxBody.buf = bytes.NewBuffer([]byte("cpu_load_short,host=server01,region=us-west value=0.64 1434055562000000000"))
	r, err := http.NewRequest(http.MethodPost, ValidServer.URL, influxBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.handleStandard(w, r, ti)
	WriterTest(t, InfluxParsePointWriter, w)
	h.backends = h.backends[:0]
}

func TestAdmin(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)

	adminBody.buf = new(bytes.Buffer)
	r, err := http.NewRequest(http.MethodPost, ValidServer.URL, adminBody)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: ValidServer.URL + "/influxdb", Admin: ValidServer.URL}

	b, _ := newHTTPBackend(&cfg)
	h.backends = append(h.backends, b)
	h.handleAdmin(w, r, ti)
	WriterTest(t, AdminWriter, w)
}

func TestAdminNoBackends(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)

	adminBody.buf = new(bytes.Buffer)
	r, err := http.NewRequest(http.MethodPost, ValidServer.URL, adminBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handleAdmin(w, r, ti)
	WriterTest(t, AdminWriterNoBackEnds, w)
}

func TestAdminNoAdminBackends(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)

	adminBody.buf = new(bytes.Buffer)
	r, err := http.NewRequest(http.MethodPost, ValidServer.URL, adminBody)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: ValidServer.URL + "/influxdb", Admin: ""}

	b, _ := newHTTPBackend(&cfg)
	h.backends = append(h.backends, b)
	h.handleAdmin(w, r, ti)
	WriterTest(t, AdminWriterNoAdminBackEnds, w)
}

func TestAdminBadMethod(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)

	adminBody.buf = new(bytes.Buffer)
	r, err := http.NewRequest(http.MethodDelete, ValidServer.URL, adminBody)
	if err != nil {
		t.Fatal(err)
	}
	h.handleAdmin(w, r, ti)
	WriterTest(t, AdminWriterWrongMethod, w)
}

func TestAdminBadURL(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)

	adminBody.buf = new(bytes.Buffer)
	r, err := http.NewRequest(http.MethodPost, ValidServer.URL, adminBody)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: ValidServer.URL + "/influxdb", Admin: "127.0.0.1:8888"}

	b, _ := newHTTPBackend(&cfg)
	h.backends = append(h.backends, b)
	h.handleAdmin(w, r, ti)
	WriterTest(t, AdminWriterWrongURL, w)
}

func TestAdminUnreachableURL(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)

	adminBody.buf = new(bytes.Buffer)
	r, err := http.NewRequest(http.MethodPost, ValidServer.URL, adminBody)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: ValidServer.URL + "/influxdb", Admin: "http://invalid/"}

	b, _ := newHTTPBackend(&cfg)
	h.backends = append(h.backends, b)
	h.handleAdmin(w, r, ti)
	WriterTest(t, AdminWriterUnreachableURL, w)
}

func TestAdminErrorServer(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)

	adminBody.buf = new(bytes.Buffer)
	r, err := http.NewRequest(http.MethodPost, Error500.URL, adminBody)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: Error500.URL + "/influxdb", Admin: Error500.URL}

	b, _ := newHTTPBackend(&cfg)
	h.backends = append(h.backends, b)

	h.handleAdmin(w, r, ti)
	WriterTest(t, AdminWriterServerError, w)
}

func TestAdminErrorClient(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, emptyConfig, false)

	adminBody.buf = new(bytes.Buffer)
	r, err := http.NewRequest(http.MethodPost, Error400.URL, adminBody)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: Error400.URL + "/influxdb", Admin: Error400.URL}

	b, _ := newHTTPBackend(&cfg)
	h.backends = append(h.backends, b)

	h.handleAdmin(w, r, ti)
	buf, _ := ioutil.ReadAll(w.writeBuf)
	buf2, _ := ioutil.ReadAll(AdminWriterClientError.writeBuf)
	assert.Equal(t, buf[:43], buf2[:43])
}
