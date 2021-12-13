package service

// TODO(mna): delete after integration tests added

/*
func TestDecodeCreateLabelRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/labels", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeCreateLabelRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(createLabelRequest)
		assert.Equal(t, "foo", *params.payload.Name)
		assert.Equal(t, "select * from foo;", *params.payload.Query)
		assert.Equal(t, "darwin", *params.payload.Platform)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
        "name": "foo",
        "query": "select * from foo;",
		"platform": "darwin"
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/v1/fleet/labels", &body),
	)
}
*/
