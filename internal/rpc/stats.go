package rpc

import (
	"fmt"
	"net/http"

	"github.com/moson-mo/goaurrpc/internal/consts"
)

const statsHtml = `
<html>
<pre>
<b>goaurrpc</b><br/>
version:			%s
last refresh:			%s
number of packages:		%d
</pre>
<html/>
`

func (s *server) handleStats(w http.ResponseWriter, r *http.Request) {
	ip := getRealIP(r, s.conf.TrustedReverseProxies)
	s.LogVeryVerbose("Client connected:", ip, "->", "["+r.Method+"]", r.URL)
	w.Header().Add("Content-Type", consts.ContentTypeHtml)
	s.mut.RLock()
	defer s.mut.RUnlock()
	np := len(s.memDB.PackageSlice)
	lr := s.lastRefresh.UTC().Format("2006-01-02 - 15:04:05 (UTC)")
	fmt.Fprintf(w, statsHtml, s.ver, lr, np)
}
