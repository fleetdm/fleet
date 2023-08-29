package main

import (
	"fmt"
	"net/http"
)

// TokenHandler return an STS Token
func TokenHandler(w http.ResponseWriter, r *http.Request) {
	// Print querystring

	if r.Method == http.MethodGet {
		fmt.Printf("====================Query String GET:\n%s\n====================", r.URL.RawQuery)
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		w.Write([]byte(`
				<h3>MDM Federated Login</h3>
								
				<script>
				function performPost() {
				  var form = document.createElement('form');
				  form.method = 'POST';
				  form.action = "` + r.URL.Query().Get("appru") + `"

				  // Add any form fields or data you want to send
				  var input1 = document.createElement('input');
				  input1.type = 'hidden';
				  input1.name = 'wresult';
				  input1.value = 'test magic';
				  form.appendChild(input1);

				  // Submit the form
				  document.body.appendChild(form);
				  form.submit();
				}


				// Call performPost() when the script is executed
				performPost();
			  	</script>
				`))

		return
	} else if r.Method == http.MethodPost {
		fmt.Printf("====================Query String POST:\n%s\n====================", r.URL.RawQuery)
	}
}
