source = ["./dist/macos_darwin_amd64/orbit"]
bundle_id = "com.fleetdm.orbit"

apple_id {
  username = "zach@fleetdm.com"
  password = "@env:AC_PASSWORD"
}

sign {
  application_identity = "D208111AA5D441DE07993F833E1F36F67526F489"
}

