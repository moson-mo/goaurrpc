package rpc

import (
	"fmt"
	"net/http"
)

const info = `
<html>
<pre>
<b>goaurrpc</b><br/>
version:	%s
last refresh:	%s
</pre>
<html/>
`

func (s *server) rpcInfoHandler(w http.ResponseWriter, r *http.Request) {
	ip := getRealIP(r, s.settings.TrustedReverseProxies)
	s.LogVerbose("Client connected:", ip, "->", "["+r.Method+"]", r.URL)
	w.Header().Add("Content-Type", "text/html; charset=UTF-8")
	s.mut.RLock()
	defer s.mut.RUnlock()
	lr := s.lastmod
	if lr == "" {
		lr = s.lastRefresh.UTC().Format("2006-01-02 - 15:04:05 (UTC)")
	}
	fmt.Fprintf(w, info, s.ver, lr)
}
