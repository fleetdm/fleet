package main

import (
	"bytes"
	"net/http"
	"strconv"
	"text/template"
)

// TOSHandler is the HTTP handler assosiated with the Terms Of Use endpoint
func TOSHandler(w http.ResponseWriter, r *http.Request) {

	// Get query param named test
	//test := r.URL.Query().Get("test")

	//RequestURI: "/TOS.svc?api-version=1.0&redirect_uri=ms-appx-web%3a%2f%2fMicrosoft.AAD.BrokerPlugin&client-request-id=bbd77af5-3c4d-4b4e-aef6-6360a94ffb93"

	redirectUri := r.URL.Query().Get("redirect_uri")
	clientReqID := r.URL.Query().Get("client-request-id")

	tmpl, err := template.New("").Parse(`
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta http-equiv="X-UA-Compatible" content="IE11">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Document</title>
	
		<style>
			html, body, #root {
				width: 100%;
				height: 100%;
				margin: 0;
				padding: 0;
				overflow: hidden;
			}
		</style>
	</head>
	<body>
    <h1>PDF Example by Object Tag</h1>
    <object data="https://marcoslabs.org/static/sample.pdf" type="application/pdf" width="100%" height="500px">
      <p>Unable to display PDF file. <a href="https://marcoslabs.org/static/sample.pdf">Download</a> instead.</p>
    </object>	
	<iframe src="https://marcoslabs.org/static/sample.pdf" frameborder="0" height="100%" width="100%">
	</iframe>		
	</body>
	</html>`)
	if err != nil {
		return
	}

	/*
	   	tmpl, err := template.New("").Parse(`
	   	<html>
	   	<script type='text/javascript'>
	   	var redirectURL = 'https://viewerjs.org/examples/';
	   	window.location = redirectURL;
	   	</script>
	   	<body>
	   	Redirecting to Fleet ...
	   	</body>
	   	</html>
	      `)
	   	if err != nil {
	   		return
	   	}
	*/
	var htmlBuf bytes.Buffer
	err = tmpl.Execute(&htmlBuf, map[string]string{"RedirectURL": redirectUri, "ClientData": clientReqID})
	if err != nil {
		return
	}

	// Return response body
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(htmlBuf.String())))
	w.Write(htmlBuf.Bytes())
}
