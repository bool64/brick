// Package zpages provides OpenCensus zpages handlers.
package zpages

import (
	"bytes"
	"html"
	"net/http"
	"net/http/httptest"
	"regexp"

	"go.opencensus.io/zpages"
)

const css = `
body{font-family: 'Roboto',sans-serif;
font-size: 14px;background-color: #F2F4EC;}
h1{color: #3D3D3D;text-align: center;margin-bottom: 20px;}
p{padding: 0 0.5em;color: #3D3D3D;}
h2{color: #3D3D3D;font-size: 1.5em;background-color: #FFF;
line-height: 2.0;margin-bottom: 0;padding: 0 0.5em;}
h3{font-size:16px;padding:0 0.5em;margin-top:6px;margin-bottom:25px;}
a{color:#A94442;}
p.header{font-family: 'Open Sans', sans-serif;top: 0;left: 0;width: 100%;
height: 60px;vertical-align: middle;color: #C1272D;font-size: 22pt;}
p.view{font-size: 20px;margin-bottom: 0;}
.header span{color: #3D3D3D;}
img.oc{vertical-align: middle;}
table{width: 100%;color: #FFF;background-color: #FFF;overflow: hidden;
margin-bottom: 30px;margin-top: 0;border-bottom: 1px solid #3D3D3D;
border-left: 1px solid #3D3D3D;border-right: 1px solid #3D3D3D;}
table.title{width:100%;color:#3D3D3D;background-color:#FFF;
border:none;line-height:2.0;margin-bottom:0;}
thead{color: #FFF;background-color: #A94442;
line-height:3.0;padding:0 0.5em;}
th{color: #FFF;background-color: #A94442;
line-height:3.0;padding:0 0.5em;}
th.borderL{border-left:1px solid #FFF; text-align:left;}
th.borderRL{border-right:1px solid #FFF; text-align:left;}
th.borderLB{border-left:1px solid #FFF;
border-bottom:1px solid #FFF;margin:0 10px;}
tr.direct{font-size:16px;padding:0 0.5em;background-color:#F2F4EC;}
tr:nth-child(even){background-color: #F2F2F2;}
td{color: #3D3D3D;line-height: 2.0;text-align: left;padding: 0 0.5em;}
td.borderLC{border-left:1px solid #3D3D3D;text-align:center;}
td.borderLL{border-left:1px solid #3D3D3D;text-align:left;}
td.borderRL{border-right:1px solid #3D3D3D;text-align:left;}
td.borderRW{border-right:1px solid #FFF}
td.borderLW{border-left:1px solid #FFF;}
td.centerW{text-align:center;color:#FFF;}
td.center{text-align:center;color:#3D3D3D;}
tr.bgcolor{background-color:#A94442;}
h1.left{text-align:left;margin-left:20px;}
table.small{width:40%;background-color:#FFF;
margin-left:20px;margin-bottom:30px;}
table.small{width:40%;background-color:#FFF;
margin-left:20px;margin-bottom:30px;}
td.col_headR{background-color:#A94442;
line-height:3.0;color:#FFF;border-right:1px solid #FFF;}
td.col_head{background-color:#A94442;
line-height:3.0;color:#FFF;}
b.title{margin-left:20px;font-weight:bold;line-height:2.0;}
input.button{margin-left:20px;margin-top:4px;
font-size:20px;width:80px;height:60px;}
td.head{text-align:center;color:#FFF;line-height:3.0;}`

// Mux creates zpages mux to serve at prefixed path.
// If traceToURL is not nil, sampled traces are converted to URLs (URL could
// lead to Jaeger instance for example).
func Mux(prefix string, traceToURL func(traceID string) string) http.Handler {
	mux := http.NewServeMux()
	zpages.Handle(mux, prefix+"/")

	sampledTraces := regexp.MustCompile(`<b style="color:blue">([a-z0-9]{32})</b>`)

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// hijacking css
		if req.RequestURI == prefix+"/public/opencensus.css" {
			rw.Header().Set("Content-Type", "text/css; charset=utf-8")
			_, err := rw.Write([]byte(css))
			if err != nil {
				panic(err)
			}

			return
		}

		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		body := bytes.Replace(
			w.Body.Bytes(),
			[]byte(`//www.opencensus.io/favicon.ico`),
			[]byte(`https://opencensus.io/images/favicon.ico`),
			1,
		)

		if traceToURL != nil {
			matches := sampledTraces.FindAllStringSubmatch(string(body), -1)
			for _, m := range matches {
				url := traceToURL(m[1])
				body = bytes.Replace(body, []byte(m[1]), []byte(`<a href="`+html.EscapeString(url)+`">`+m[1]+`</a>`), 1)
			}
		}

		_, err := rw.Write(body)
		if err != nil {
			panic(err)
		}
	})
}
