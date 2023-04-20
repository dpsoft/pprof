package pprof

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	rpprof "runtime/pprof"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultPrefix url prefix of pprof
	DefaultPrefix = "/debug/pprof"
)

func getPrefix(prefixOptions ...string) string {
	prefix := DefaultPrefix
	if len(prefixOptions) > 0 {
		prefix = prefixOptions[0]
	}
	return prefix
}

// Register the standard HandlerFuncs from the net/http/pprof package with
// the provided gin.Engine. prefixOptions is a optional. If not prefixOptions,
// the default path prefix is used, otherwise first prefixOptions will be path prefix.
func Register(r *gin.Engine, prefixOptions ...string) {
	RouteRegister(&(r.RouterGroup), prefixOptions...)
}

// RouteRegister the standard HandlerFuncs from the net/http/pprof package with
// the provided gin.GrouterGroup. prefixOptions is a optional. If not prefixOptions,
// the default path prefix is used, otherwise first prefixOptions will be path prefix.
func RouteRegister(rg *gin.RouterGroup, prefixOptions ...string) {
	prefix := getPrefix(prefixOptions...)

	prefixRouter := rg.Group(prefix)
	{
		prefixRouter.GET("/", gin.WrapF(pprof.Index))
		prefixRouter.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		prefixRouter.GET("/profile", gin.WrapF(Profile))
		prefixRouter.POST("/symbol", gin.WrapF(pprof.Symbol))
		prefixRouter.GET("/symbol", gin.WrapF(pprof.Symbol))
		prefixRouter.GET("/trace", gin.WrapF(pprof.Trace))
		prefixRouter.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
		prefixRouter.GET("/block", gin.WrapH(pprof.Handler("block")))
		prefixRouter.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		prefixRouter.GET("/heap", gin.WrapH(pprof.Handler("heap")))
		prefixRouter.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
		prefixRouter.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
	}
}

func Profile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	sec, err := strconv.ParseInt(r.FormValue("seconds"), 10, 64)
	if sec <= 0 || err != nil {
		sec = 3
	}

	rate, err := strconv.ParseInt(r.FormValue("rate"), 10, 64)
	if rate < 0 || err != nil {
		rate = 49
	}

	//set cpu profile rate to 49 hz
	runtime.SetCPUProfileRate(int(rate))

	if durationExceedsWriteTimeout(r, float64(sec)) {
		serveError(w, http.StatusBadRequest, "profile duration exceeds server's WriteTimeout")
		return
	}

	// Set Content Type assuming StartCPUProfile will work,
	// because if it does it starts writing.
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="profile"`)
	if err := rpprof.StartCPUProfile(w); err != nil {
		// StartCPUProfile failed, so no writes yet.
		serveError(w, http.StatusInternalServerError,
			fmt.Sprintf("Could not enable CPU profiling: %s", err))
		return
	}
	sleep(r, time.Duration(sec)*time.Second)
	rpprof.StopCPUProfile()
}

func serveError(w http.ResponseWriter, status int, txt string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Go-Pprof", "1")
	w.Header().Del("Content-Disposition")
	w.WriteHeader(status)
	fmt.Fprintln(w, txt)
}

func durationExceedsWriteTimeout(r *http.Request, seconds float64) bool {
	srv, ok := r.Context().Value(http.ServerContextKey).(*http.Server)
	return ok && srv.WriteTimeout != 0 && seconds >= srv.WriteTimeout.Seconds()
}

func sleep(r *http.Request, d time.Duration) {
	select {
	case <-time.After(d):
	case <-r.Context().Done():
	}
}
