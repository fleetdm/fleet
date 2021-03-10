source = ["./dist/macos_darwin_amd64/orbit"]
bundle_id = "com.fleetdm.orbit"

apple_id {
  username = "@env:AC_USERNAME"
  password = "@env:AC_PASSWORD"
}

sign {
  application_identity = "51049B247B25B3119FAE7E9C0CC4375A43E47237"
}

