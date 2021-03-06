package server_test

import (
	"testing"

	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/leon332157/replish/server"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ghttp "github.com/onsi/gomega/ghttp"
	log "github.com/sirupsen/logrus"

	"github.com/valyala/fasthttp"
)
// TODO: BROKEN TEST
func UNUSED(x ...interface{}) {

}

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Replish Server")
}

var _ = BeforeSuite(func() {
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	log.SetReportCaller(true)
	log.SetLevel(log.ErrorLevel)
	startGhttpServer()
	_, rawPort, _ := net.SplitHostPort(ghttpServer.Addr())
	intPort, _ := strconv.Atoi(rawPort)
	go server.StartForwardServer(uint16(intPort))
	go server.StartMain(8484, uint16(intPort))
	time.Sleep(2 * time.Second)})
var _ = AfterSuite(func() {
	ghttpServer.Close()
})

var client = &fasthttp.Client{}
var ghttpServer = ghttp.NewUnstartedServer()

func startGhttpServer() {
	fmt.Printf("GHTTP addr: %s\n", ghttpServer.Addr())
	ghttpServer.RouteToHandler("GET", "/", ghttp.CombineHandlers(ghttp.VerifyRequest("GET", "/"), ghttp.RespondWith(http.StatusOK, "test")))
	ghttpServer.RouteToHandler("POST", "/post", ghttp.CombineHandlers(ghttp.VerifyRequest("POST", "/post"), ghttp.RespondWith(http.StatusOK, "test")))
	go ghttpServer.Start()
}

var _ = Describe("Replish Server", func() {

	XDescribe("TCP Forwarder", func() {
		It("should serve 10000 requests (POST & GET)", func() {
			Expect(makeRequests(10000, 8383)).To(Succeed())

		})
		XMeasure("1000 requests with 10 samples (POST & GET)", func(b Benchmarker) {
			b.Time("runtime", func() { makeRequests(1000, 8383) })
		}, 10)
	})
	Describe("Reverse Proxy", func() {
		It("should serve 10000 requests with 10 samples (POST & GET)", func() {
			Expect(makeRequests(10000, 8484)).To(Succeed())
		})
		XMeasure("1000 requests with 10 samples (POST & GET)", func(b Benchmarker) {
			b.Time("runtime", func() { makeRequests(1000, 8484) })
		}, 10)
	})
})

func makeRequests(n int, port int) error {
	url := fmt.Sprintf("http://127.0.0.1:%v", port)
	var (
		req  fasthttp.Request
		resp fasthttp.Response
	)
	for x := 0; x < n; x++ {
		req.SetRequestURI(url)
		req.Header.SetMethod("GET")
		//fmt.Printf("%s\n", req.RequestURI())
		err := client.DoTimeout(&req, &resp, 500*time.Millisecond)
		if err != nil {
			return fmt.Errorf("Failed on attempt %v err: %v", x, err)
		}
		if resp.StatusCode() != fasthttp.StatusOK {
			return fmt.Errorf("Unexpected status code: %d. Expecting %d", resp.StatusCode(), fasthttp.StatusOK)
		}
		// Assuming GET didn't fail, POST shouldn't fail either.
		req.SetRequestURI(url + "/post") // switch URI to post
		req.Header.SetMethod("POST")
		//fmt.Printf("%s\n", req.RequestURI())
		err = client.DoTimeout(&req, &resp, 1*time.Second)
		if err != nil {
			return fmt.Errorf("Failed on attempt %v err: %v", x, err)
		}
		if resp.StatusCode() != fasthttp.StatusOK {
			return fmt.Errorf("Unexpected status code: %d. Expecting %d", resp.StatusCode(), fasthttp.StatusOK)
		}
	}
	return nil
}
