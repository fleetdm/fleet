# remediate-emudhra-ca-compromise-windows.ps1
# Remove trust in the compromised eMudhra/emSign intermediates (Oct 2026).
#   1. Adds all 16 compromised intermediate certificates (8 keys, each issued by emSign
#      and cross-signed by Sectigo AAA) to LocalMachine\Disallowed ("Untrusted Certificates").
#      Works even where Microsoft's automatic disallowed-list update is blocked or offline.
#   2. Strips the two compromised roots from fleetd's CA bundle
#      (C:\Program Files\Orbit\certs.pem), which osqueryd uses for its TLS connection to Fleet.
#      Only after proving the Fleet server's chain still reaches a self-signed root in the bundle.
# Idempotent: safe to re-run. Built for Fleet "Run script" (runs as SYSTEM).
# Save it as a Fleet script (it is larger than the 10,000-character ad hoc limit).
# Rollback: copy certs.pem.pre-emudhra back, and delete the 16 entries from
# "Untrusted Certificates" in certlm.msc.
# Exit: 0 = protected, 2 = a step was skipped or needs attention (read output), 1 = error.
# Source: https://emudhra.we-have-a-problem.com/

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2

# SHA-1 of: emSign ECC Root CA - G3, AAA Certificate Services
$BlockedRootSha1 = @('3043FA4FF257DCA0C380EE2E58EA78B23FE6BBC1', 'D1EB23A46D17D68FD92564C2F1F1601764D8E349')

$Intermediates = @(
    @{ Name = 'EM DV TLS CA - G3B (via emSign)'; Sha1 = 'A99B289464F3B44343270A7A1E910F02370483FE'; B64 = 'MIIDaDCCAu6gAwIBAgIQAKSu5njyM2K0wORgt3LyFDAKBggqhkjOPQQDAzBWMQswCQYDVQQGEwJJTjElMCMGA1UECgwcZU11ZGhyYSBUZWNobm9sb2dpZXMgTGltaXRlZDEgMB4GA1UEAwwXZW1TaWduIFJvb3QgVExTIENBIC0gRzMwHhcNMjUwNDMwMTU1NDA4WhcNNDAwNDI4MTgzMDAwWjA/MQswCQYDVQQGEwJOTDETMBEGA1UECgwKZU11ZGhyYSBCVjEbMBkGA1UEAwwSRU0gRFYgVExTIENBIC0gRzNCMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAE557ZwEqXc8RzXlERHahawr1eZKs3S4DhLLwgOOweTw47gS/JhZ4eZ6+wTi+LyX90hbplqpEe6KTkW2LVzAPVa+Z4s8OMtNyTfKJGDiOEU8iSk10p1E+R4VYGtjzO2rRoo4IBljCCAZIwHwYDVR0jBBgwFoAU4TqrIXzx401UrtRyt/2LWUIgkFQwHQYDVR0OBBYEFOIeL+ccGTkOVUtiwSAkVzUkiiUUMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjBdBgNVHSAEVjBUMDkGCysGAQQBg44hAQABMCowKAYIKwYBBQUHAgEWHGh0dHA6Ly9yZXBvc2l0b3J5LmVtc2lnbi5jb20wDQYLKwYBBAGDjiEBAmQwCAYGZ4EMAQIBMHUGCCsGAQUFBwEBBGkwZzAkBggrBgEFBQcwAYYYaHR0cDovL29jc3AtYi5lbXNpZ24uY29tMD8GCCsGAQUFBzAChjNodHRwOi8vcmVwb3NpdG9yeS5lbXNpZ24uY29tL2NlcnRzL0VNRFZUTFNDQUczQi5wN2MwNwYDVR0fBDAwLjAsoCqgKIYmaHR0cDovL2NybC5lbXNpZ24uY29tLz9yb290dGxzY2FnMy5jcmwwDgYDVR0PAQH/BAQDAgEGMBIGA1UdEwEB/wQIMAYBAf8CAQAwCgYIKoZIzj0EAwMDaAAwZQIwLmsGXi6oFMJ2klkcEMXMug6gvxQWZWkewiDWVzQyl2L6z6DuQfPOIbh1grvdULDCAjEAxZvLlxUavG69J6wnSjC8XBbqNIQsJ6CE2QdNOe84Ptxb6VG8gt2TlfJbHii+KNvK' },
    @{ Name = 'EM DV TLS CA - G3B (via Sectigo AAA)'; Sha1 = '1396BE1372BC93DFA2A433CA964DBA91548A6E57'; B64 = 'MIIDzjCCAragAwIBAgIRAKAk/fS6Zlx4XOSgIEg4WLcwDQYJKoZIhvcNAQELBQAwezELMAkGA1UEBhMCR0IxGzAZBgNVBAgMEkdyZWF0ZXIgTWFuY2hlc3RlcjEQMA4GA1UEBwwHU2FsZm9yZDEaMBgGA1UECgwRQ29tb2RvIENBIExpbWl0ZWQxITAfBgNVBAMMGEFBQSBDZXJ0aWZpY2F0ZSBTZXJ2aWNlczAeFw0yNTA1MTQwMDAwMDBaFw0yODEyMzEyMzU5NTlaMD8xCzAJBgNVBAYTAk5MMRMwEQYDVQQKDAplTXVkaHJhIEJWMRswGQYDVQQDDBJFTSBEViBUTFMgQ0EgLSBHM0IwdjAQBgcqhkjOPQIBBgUrgQQAIgNiAATnntnASpdzxHNeUREdqFrCvV5kqzdLgOEsvCA47B5PDjuBL8mFnh5nr7BOL4vJf3SFumWqkR7opORbYtXMA9Vr5nizw4y03JN8okYOI4RTyJKTXSnUT5HhVga2PM7atGijggE1MIIBMTAfBgNVHSMEGDAWgBSgEQojPpbxB+zirynvgqV/0DCktDAdBgNVHQ4EFgQU4h4v5xwZOQ5VS2LBICRXNSSKJRQwDgYDVR0PAQH/BAQDAgEGMBIGA1UdEwEB/wQIMAYBAf8CAQAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMDEGA1UdIAQqMCgwCAYGZ4EMAQIBMA0GCysGAQQBg44hAQABMA0GCysGAQQBg44hAQJkMEMGA1UdHwQ8MDowOKA2oDSGMmh0dHA6Ly9jcmwuY29tb2RvY2EuY29tL0FBQUNlcnRpZmljYXRlU2VydmljZXMuY3JsMDQGCCsGAQUFBwEBBCgwJjAkBggrBgEFBQcwAYYYaHR0cDovL29jc3AuY29tb2RvY2EuY29tMA0GCSqGSIb3DQEBCwUAA4IBAQBzQ1IoqpdotM6gFrR4Q3YrKifOLeaj5XXunldpVE/vZn2TxC2y0s8oYiLR5ICEwqVXSB7N6BM1Mp+Nif7Jbhb2IswAfSR4Iik8+B0iRmi7lYoNlycCgHjmsjOmZaxMQkPmjXAJbG5f4V41OsvJZmS7lI3r0IuvEDVX3GAZ13on6IImSs9ybNpHQ7NCqkMQ7wH4uAO1dtpFycgJSdHaCOhHjrox2BTYGLdpUrYwY3EOAjlZ8UGgsxs7mG+xquMlbN1LFURwfnkVMP3T/DBIZ/ys2+lNFyxK3FlArrV4/000uTnVB1lO8oGORYdXoT5VGUyRCEIDhF5TBUuKF4E3kf1h' },
    @{ Name = 'EM OV TLS CA - G3B (via emSign)'; Sha1 = '5F014E0DB3BB8F950E0598AB280846D383FC57BE'; B64 = 'MIIDZjCCAu2gAwIBAgIPbK2jZ/scEACIEg6iAtVoMAoGCCqGSM49BAMDMFYxCzAJBgNVBAYTAklOMSUwIwYDVQQKDBxlTXVkaHJhIFRlY2hub2xvZ2llcyBMaW1pdGVkMSAwHgYDVQQDDBdlbVNpZ24gUm9vdCBUTFMgQ0EgLSBHMzAeFw0yNTA0MzAxNjAwMTJaFw00MDA0MjgxODMwMDBaMD8xCzAJBgNVBAYTAk5MMRMwEQYDVQQKDAplTXVkaHJhIEJWMRswGQYDVQQDDBJFTSBPViBUTFMgQ0EgLSBHM0IwdjAQBgcqhkjOPQIBBgUrgQQAIgNiAARziXqg6zz9yosSQC/68KvE4SErr43ud4rJ7i1tuD+sKAxeBWOK56Ks1xwyKo42HN659pxDArndjXPRzGyqsswK7k4hbOl53VGKSvS2JuVoTN1lgDftDIVOTMEvTPhkbQ2jggGWMIIBkjAfBgNVHSMEGDAWgBThOqshfPHjTVSu1HK3/YtZQiCQVDAdBgNVHQ4EFgQUm9hlUa/n5yw3rQjuaRnZvzeErgowHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMF0GA1UdIARWMFQwOQYLKwYBBAGDjiEBAAEwKjAoBggrBgEFBQcCARYcaHR0cDovL3JlcG9zaXRvcnkuZW1zaWduLmNvbTANBgsrBgEEAYOOIQECbjAIBgZngQwBAgIwdQYIKwYBBQUHAQEEaTBnMCQGCCsGAQUFBzABhhhodHRwOi8vb2NzcC1iLmVtc2lnbi5jb20wPwYIKwYBBQUHMAKGM2h0dHA6Ly9yZXBvc2l0b3J5LmVtc2lnbi5jb20vY2VydHMvRU1PVlRMU0NBRzNCLnA3YzA3BgNVHR8EMDAuMCygKqAohiZodHRwOi8vY3JsLmVtc2lnbi5jb20vP3Jvb3R0bHNjYWczLmNybDAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0TAQH/BAgwBgEB/wIBADAKBggqhkjOPQQDAwNnADBkAjBAk+5BizdJhrED3wb7F2Gm7bWI60dlNd075nghT2TrR4NWKZtJZcbc28xxIY/NONICMF6mPdsjosaqmvEMWqKcBh4vsIvXSKnQj5o8zrFB4oKS+accjYTe8Dz3bQjauvy9uw==' },
    @{ Name = 'EM OV TLS CA - G3B (via Sectigo AAA)'; Sha1 = '5DFA19197D1C1CF629AF7C0CD865543C5050227C'; B64 = 'MIIDzjCCAragAwIBAgIRAOlCI73pL5FOWrmsq/2sNxAwDQYJKoZIhvcNAQELBQAwezELMAkGA1UEBhMCR0IxGzAZBgNVBAgMEkdyZWF0ZXIgTWFuY2hlc3RlcjEQMA4GA1UEBwwHU2FsZm9yZDEaMBgGA1UECgwRQ29tb2RvIENBIExpbWl0ZWQxITAfBgNVBAMMGEFBQSBDZXJ0aWZpY2F0ZSBTZXJ2aWNlczAeFw0yNTA1MTQwMDAwMDBaFw0yODEyMzEyMzU5NTlaMD8xCzAJBgNVBAYTAk5MMRMwEQYDVQQKDAplTXVkaHJhIEJWMRswGQYDVQQDDBJFTSBPViBUTFMgQ0EgLSBHM0IwdjAQBgcqhkjOPQIBBgUrgQQAIgNiAARziXqg6zz9yosSQC/68KvE4SErr43ud4rJ7i1tuD+sKAxeBWOK56Ks1xwyKo42HN659pxDArndjXPRzGyqsswK7k4hbOl53VGKSvS2JuVoTN1lgDftDIVOTMEvTPhkbQ2jggE1MIIBMTAfBgNVHSMEGDAWgBSgEQojPpbxB+zirynvgqV/0DCktDAdBgNVHQ4EFgQUm9hlUa/n5yw3rQjuaRnZvzeErgowDgYDVR0PAQH/BAQDAgEGMBIGA1UdEwEB/wQIMAYBAf8CAQAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMDEGA1UdIAQqMCgwCAYGZ4EMAQICMA0GCysGAQQBg44hAQABMA0GCysGAQQBg44hAQJuMEMGA1UdHwQ8MDowOKA2oDSGMmh0dHA6Ly9jcmwuY29tb2RvY2EuY29tL0FBQUNlcnRpZmljYXRlU2VydmljZXMuY3JsMDQGCCsGAQUFBwEBBCgwJjAkBggrBgEFBQcwAYYYaHR0cDovL29jc3AuY29tb2RvY2EuY29tMA0GCSqGSIb3DQEBCwUAA4IBAQALRoOSvkzRa+tN5Rje9W5GiXd/qsicd9N6Gnh0SWC7VyWH+HycIqAnFCMYZznjCeGusomFYHE42SHWr85JDIbgzy6y3dO4i9n9w0Bx7yrXWBbDzFS8VgA3+EY4Cg7ymtoC3RPOJVtaXXPZ+68X804g1Mj+3vCSHj2lNPVgco0OzSIOJ/KMg1R9MLBkiKzbbERAGZHejY0HBhYd76IINZN8z6bBNah2UdFYxsOisLynPQUkol6gRLum024/FzVlCCDfnUusrkr+mAfPAvD/Vb8bhpA7jYtepuU+2mIdShj5lLlvAwZphn+ypWycuo2x0k9mldz2dqI9nhCiY+TTIOOR' },
    @{ Name = 'EM EV TLS CA - G3B (via emSign)'; Sha1 = 'F33ACD1F5D31AA81AC18C72C928446226D3D5914'; B64 = 'MIIDaDCCAu2gAwIBAgIQAMfEHv6S2CXwXq6EGkqCwTAKBggqhkjOPQQDAzBWMQswCQYDVQQGEwJJTjElMCMGA1UECgwcZU11ZGhyYSBUZWNobm9sb2dpZXMgTGltaXRlZDEgMB4GA1UEAwwXZW1TaWduIFJvb3QgVExTIENBIC0gRzMwHhcNMjUwNDMwMTYwMzMxWhcNNDAwNDI4MTgzMDAwWjA/MQswCQYDVQQGEwJOTDETMBEGA1UECgwKZU11ZGhyYSBCVjEbMBkGA1UEAwwSRU0gRVYgVExTIENBIC0gRzNCMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEJduoWmTTl9oLEFLp6lRuSHvv6pydnWigxerc/u91ppu11vwB9Cv7ym60C81kwHZ+Nv/QXZeoE8WqDGyVMOec9fCGZGjxOAZCCqXIUYQLHiUlQOeQ5LZUw4Uu1krx6GV9o4IBlTCCAZEwHwYDVR0jBBgwFoAU4TqrIXzx401UrtRyt/2LWUIgkFQwHQYDVR0OBBYEFD7cINmJRpiHe4c/2IFHslwV2EcGMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjBcBgNVHSAEVTBTMDkGCysGAQQBg44hAQABMCowKAYIKwYBBQUHAgEWHGh0dHA6Ly9yZXBvc2l0b3J5LmVtc2lnbi5jb20wDQYLKwYBBAGDjiEBAngwBwYFZ4EMAQEwdQYIKwYBBQUHAQEEaTBnMCQGCCsGAQUFBzABhhhodHRwOi8vb2NzcC1iLmVtc2lnbi5jb20wPwYIKwYBBQUHMAKGM2h0dHA6Ly9yZXBvc2l0b3J5LmVtc2lnbi5jb20vY2VydHMvRU1FVlRMU0NBRzNCLnA3YzA3BgNVHR8EMDAuMCygKqAohiZodHRwOi8vY3JsLmVtc2lnbi5jb20vP3Jvb3R0bHNjYWczLmNybDAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0TAQH/BAgwBgEB/wIBADAKBggqhkjOPQQDAwNpADBmAjEA9Jq999oaOUKhFwopw33iCmt0OHIn3gjT43GfuOHR/YckXdwoNdDAlIEhz0NgRO3/AjEAvY/a3OMvktIZLYjn+WLKmp0QhrIMV+e6EpS7j4GVeOKouNp6MN/P+ByGeiRMghzB' },
    @{ Name = 'EM EV TLS CA - G3B (via Sectigo AAA)'; Sha1 = '19069BD6533BFF93FB67C71FB3376B4EA7402C3B'; B64 = 'MIIDzTCCArWgAwIBAgIRANj4aF5PK9nYJAKoWhwetzEwDQYJKoZIhvcNAQELBQAwezELMAkGA1UEBhMCR0IxGzAZBgNVBAgMEkdyZWF0ZXIgTWFuY2hlc3RlcjEQMA4GA1UEBwwHU2FsZm9yZDEaMBgGA1UECgwRQ29tb2RvIENBIExpbWl0ZWQxITAfBgNVBAMMGEFBQSBDZXJ0aWZpY2F0ZSBTZXJ2aWNlczAeFw0yNTA1MTQwMDAwMDBaFw0yODEyMzEyMzU5NTlaMD8xCzAJBgNVBAYTAk5MMRMwEQYDVQQKDAplTXVkaHJhIEJWMRswGQYDVQQDDBJFTSBFViBUTFMgQ0EgLSBHM0IwdjAQBgcqhkjOPQIBBgUrgQQAIgNiAAQl26haZNOX2gsQUunqVG5Ie+/qnJ2daKDF6tz+73Wmm7XW/AH0K/vKbrQLzWTAdn42/9Bdl6gTxaoMbJUw55z18IZkaPE4BkIKpchRhAseJSVA55DktlTDhS7WSvHoZX2jggE0MIIBMDAfBgNVHSMEGDAWgBSgEQojPpbxB+zirynvgqV/0DCktDAdBgNVHQ4EFgQUPtwg2YlGmId7hz/YgUeyXBXYRwYwDgYDVR0PAQH/BAQDAgEGMBIGA1UdEwEB/wQIMAYBAf8CAQAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMDAGA1UdIAQpMCcwBwYFZ4EMAQEwDQYLKwYBBAGDjiEBAAEwDQYLKwYBBAGDjiEBAngwQwYDVR0fBDwwOjA4oDagNIYyaHR0cDovL2NybC5jb21vZG9jYS5jb20vQUFBQ2VydGlmaWNhdGVTZXJ2aWNlcy5jcmwwNAYIKwYBBQUHAQEEKDAmMCQGCCsGAQUFBzABhhhodHRwOi8vb2NzcC5jb21vZG9jYS5jb20wDQYJKoZIhvcNAQELBQADggEBAJNFuAFW9Gnw/8+kCO4KIlZdnM1Sh2u+8cK0EYAHhXpb5FlGvYkcARigo5oS7VoM7FOYlTueV6yICZKNAYawZPJPhNCGK+rp3cQBEz6d0n8OZEYZov9lb3ll8ORgSDufFVIW6Gagg/UnOamLBTOR8eD5V73SCme/B2cHUDw+m5E8cc+0Ujzz3Cn3ex5JrhXk2T3s8PoVzEmhTgGboFfcWO1EYhPPrYwdxPYRQIXAXUXhLEOVNL9ASbZJLFI8E1IMm7NwxVN2s38jWpaCUaaF0tg7hfiy7oe13rzJfHsdqldJVTTUTPWVx8TC4ju0VLetbAlwhwFT6AP8uIVMGUHNw24=' },
    @{ Name = 'EM DV TLS CA - G3C (via emSign)'; Sha1 = '8AD23BC78845D6FAAE1D534BAC89F0E36BA55A11'; B64 = 'MIIDaDCCAu6gAwIBAgIPWIsuPCu4zaE+GhIJJUMhMAoGCCqGSM49BAMDMFYxCzAJBgNVBAYTAklOMSUwIwYDVQQKDBxlTXVkaHJhIFRlY2hub2xvZ2llcyBMaW1pdGVkMSAwHgYDVQQDDBdlbVNpZ24gUm9vdCBUTFMgQ0EgLSBHMzAeFw0yNTA0MzAxNTM1MDlaFw00MDA0MjgxODMwMDBaMEAxCzAJBgNVBAYTAlVTMRQwEgYDVQQKDAtlTXVkaHJhIEluYzEbMBkGA1UEAwwSRU0gRFYgVExTIENBIC0gRzNDMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEOnp4YsjETTJN3WkDOmeCwO/u/kBV5ooSP6A3G6O9kZRRTuFypDPjYw8CvRiAN9nlFJ+bd9uUooHfmHsv6oiLdL3+j9pMEJnubKGVZv4+8OX2sJpVbMyLpQVh3xsqRB0Bo4IBljCCAZIwHwYDVR0jBBgwFoAU4TqrIXzx401UrtRyt/2LWUIgkFQwHQYDVR0OBBYEFK5+EFw8KZbBqEcnBVnx4D3U5QrzMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjBdBgNVHSAEVjBUMDkGCysGAQQBg44hAQABMCowKAYIKwYBBQUHAgEWHGh0dHA6Ly9yZXBvc2l0b3J5LmVtc2lnbi5jb20wDQYLKwYBBAGDjiEBAmQwCAYGZ4EMAQIBMHUGCCsGAQUFBwEBBGkwZzAkBggrBgEFBQcwAYYYaHR0cDovL29jc3AtYy5lbXNpZ24uY29tMD8GCCsGAQUFBzAChjNodHRwOi8vcmVwb3NpdG9yeS5lbXNpZ24uY29tL2NlcnRzL0VNRFZUTFNDQUczQy5wN2MwNwYDVR0fBDAwLjAsoCqgKIYmaHR0cDovL2NybC5lbXNpZ24uY29tLz9yb290dGxzY2FnMy5jcmwwDgYDVR0PAQH/BAQDAgEGMBIGA1UdEwEB/wQIMAYBAf8CAQAwCgYIKoZIzj0EAwMDaAAwZQIxAMEN2HWMocbuWdy1d33pH6JpMvP/V/DIS5PtZbv1Q7TXEbxrcyxgyuddOJscYDJJlAIwB4sH1BpbScDzkdD4tJBSwrG/1Mx82vb8YaySehNuu1+1oXNdAEKwaJ0mVzScy2+6' },
    @{ Name = 'EM DV TLS CA - G3C (via Sectigo AAA)'; Sha1 = '4B0944C14F8C2CCBA4E4F32104AD920FA15EE331'; B64 = 'MIIDzzCCAregAwIBAgIRAJHzR/Glt9OZYrleDuTimdEwDQYJKoZIhvcNAQELBQAwezELMAkGA1UEBhMCR0IxGzAZBgNVBAgMEkdyZWF0ZXIgTWFuY2hlc3RlcjEQMA4GA1UEBwwHU2FsZm9yZDEaMBgGA1UECgwRQ29tb2RvIENBIExpbWl0ZWQxITAfBgNVBAMMGEFBQSBDZXJ0aWZpY2F0ZSBTZXJ2aWNlczAeFw0yNTA1MTQwMDAwMDBaFw0yODEyMzEyMzU5NTlaMEAxCzAJBgNVBAYTAlVTMRQwEgYDVQQKDAtlTXVkaHJhIEluYzEbMBkGA1UEAwwSRU0gRFYgVExTIENBIC0gRzNDMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEOnp4YsjETTJN3WkDOmeCwO/u/kBV5ooSP6A3G6O9kZRRTuFypDPjYw8CvRiAN9nlFJ+bd9uUooHfmHsv6oiLdL3+j9pMEJnubKGVZv4+8OX2sJpVbMyLpQVh3xsqRB0Bo4IBNTCCATEwHwYDVR0jBBgwFoAUoBEKIz6W8Qfs4q8p74Klf9AwpLQwHQYDVR0OBBYEFK5+EFw8KZbBqEcnBVnx4D3U5QrzMA4GA1UdDwEB/wQEAwIBBjASBgNVHRMBAf8ECDAGAQH/AgEAMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAxBgNVHSAEKjAoMAgGBmeBDAECATANBgsrBgEEAYOOIQEAATANBgsrBgEEAYOOIQECZDBDBgNVHR8EPDA6MDigNqA0hjJodHRwOi8vY3JsLmNvbW9kb2NhLmNvbS9BQUFDZXJ0aWZpY2F0ZVNlcnZpY2VzLmNybDA0BggrBgEFBQcBAQQoMCYwJAYIKwYBBQUHMAGGGGh0dHA6Ly9vY3NwLmNvbW9kb2NhLmNvbTANBgkqhkiG9w0BAQsFAAOCAQEAr7b6Mx11TIJgVzHt6EH5qGNWeiiqoMrp8E5/k8s+jvupx1n083pZ5XOzUzNDwEUe0W4C4FIGUHNQ4XDwqOw0yK42xqm2dvCKgsko2UvDlT0haZVtTCeWMQOe5J/JziaMfZwsZNfZ3aaI6XPtWDZtwY8AaNfyaeCpbtm5GcGnyNWk4jSNpbtpo6NjfXuPCFOm3rTjJ4zytP7Tso6iMm0L01cYFOlTmoPv4515R2r3SPnKDtijoB/D/vgxzojvDafgyG9afX8+laO9NbMMz4sqC1vZx3984i/sBNFiugm3Vvdq5OdJRjuGC1QILIs6dg/K3i8VQqRpbfaM9kIlMzZ3dQ==' },
    @{ Name = 'EM OV TLS CA - G3C (via emSign)'; Sha1 = '7FCC31AAB874CE6C8EE8584F54CFD08646B54117'; B64 = 'MIIDaDCCAu6gAwIBAgIPKWkmwiejoVJjkrGUgqUEMAoGCCqGSM49BAMDMFYxCzAJBgNVBAYTAklOMSUwIwYDVQQKDBxlTXVkaHJhIFRlY2hub2xvZ2llcyBMaW1pdGVkMSAwHgYDVQQDDBdlbVNpZ24gUm9vdCBUTFMgQ0EgLSBHMzAeFw0yNTA0MzAxNTM3NDFaFw00MDA0MjgxODMwMDBaMEAxCzAJBgNVBAYTAlVTMRQwEgYDVQQKDAtlTXVkaHJhIEluYzEbMBkGA1UEAwwSRU0gT1YgVExTIENBIC0gRzNDMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEE4JRzVKskpjByKrZdzId65fnCb0LTKCspV3IrVHc/J0ViaFZfjpRIOHv1jHGPhg1ysrimGmmLhYx6KKBgatWYW3EXZGKvAnzqw5jz3kqpNztc4e+N7ulaVSfHAKycO1no4IBljCCAZIwHwYDVR0jBBgwFoAU4TqrIXzx401UrtRyt/2LWUIgkFQwHQYDVR0OBBYEFDTq6gzryQra+dDF4u2beMmHnhtsMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjBdBgNVHSAEVjBUMDkGCysGAQQBg44hAQABMCowKAYIKwYBBQUHAgEWHGh0dHA6Ly9yZXBvc2l0b3J5LmVtc2lnbi5jb20wDQYLKwYBBAGDjiEBAm4wCAYGZ4EMAQICMHUGCCsGAQUFBwEBBGkwZzAkBggrBgEFBQcwAYYYaHR0cDovL29jc3AtYy5lbXNpZ24uY29tMD8GCCsGAQUFBzAChjNodHRwOi8vcmVwb3NpdG9yeS5lbXNpZ24uY29tL2NlcnRzL0VNT1ZUTFNDQUczQy5wN2MwNwYDVR0fBDAwLjAsoCqgKIYmaHR0cDovL2NybC5lbXNpZ24uY29tLz9yb290dGxzY2FnMy5jcmwwDgYDVR0PAQH/BAQDAgEGMBIGA1UdEwEB/wQIMAYBAf8CAQAwCgYIKoZIzj0EAwMDaAAwZQIxAMSZt2xsPeumRdeSMSQ3CbzIZeQ/q7GDCWfeE1Pq+mhgA6gRM44cNVF3xJ+wsxJk+wIwFJbaDMokD8Ei6eNOiA8yUMYvfzQyngmrhZQMdPCK1SewrR794rSb9vGqxZzelQth' },
    @{ Name = 'EM OV TLS CA - G3C (via Sectigo AAA)'; Sha1 = '305FC1E253DCAB71B95D4995DAE3CB09D562F22A'; B64 = 'MIIDzzCCAregAwIBAgIRAI1aSBGTnvRH13ERzdmEmukwDQYJKoZIhvcNAQELBQAwezELMAkGA1UEBhMCR0IxGzAZBgNVBAgMEkdyZWF0ZXIgTWFuY2hlc3RlcjEQMA4GA1UEBwwHU2FsZm9yZDEaMBgGA1UECgwRQ29tb2RvIENBIExpbWl0ZWQxITAfBgNVBAMMGEFBQSBDZXJ0aWZpY2F0ZSBTZXJ2aWNlczAeFw0yNTA1MTQwMDAwMDBaFw0yODEyMzEyMzU5NTlaMEAxCzAJBgNVBAYTAlVTMRQwEgYDVQQKDAtlTXVkaHJhIEluYzEbMBkGA1UEAwwSRU0gT1YgVExTIENBIC0gRzNDMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEE4JRzVKskpjByKrZdzId65fnCb0LTKCspV3IrVHc/J0ViaFZfjpRIOHv1jHGPhg1ysrimGmmLhYx6KKBgatWYW3EXZGKvAnzqw5jz3kqpNztc4e+N7ulaVSfHAKycO1no4IBNTCCATEwHwYDVR0jBBgwFoAUoBEKIz6W8Qfs4q8p74Klf9AwpLQwHQYDVR0OBBYEFDTq6gzryQra+dDF4u2beMmHnhtsMA4GA1UdDwEB/wQEAwIBBjASBgNVHRMBAf8ECDAGAQH/AgEAMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAxBgNVHSAEKjAoMAgGBmeBDAECAjANBgsrBgEEAYOOIQEAATANBgsrBgEEAYOOIQECbjBDBgNVHR8EPDA6MDigNqA0hjJodHRwOi8vY3JsLmNvbW9kb2NhLmNvbS9BQUFDZXJ0aWZpY2F0ZVNlcnZpY2VzLmNybDA0BggrBgEFBQcBAQQoMCYwJAYIKwYBBQUHMAGGGGh0dHA6Ly9vY3NwLmNvbW9kb2NhLmNvbTANBgkqhkiG9w0BAQsFAAOCAQEAHlbhZpjhkPEAQVhPHRtBDaHqiisVD20pQ1TUeSx3l62Qe7NQ0b7vWQy7DN0iKOu7BGpP5EnIjvXbz9FhpfPJ4FhKEt3tQ51rqXz2oD8vRaryvExWPobDUz3j7HN6OzBsVMIe3PD7AAkpky0x8zMSi/ep5R9OMzybmNptbtppyuB5ytgq+h7lQdRIP+CQTaJX3qTgfMqzntYre5GljmEB5hU2/rSn8zoSU2APkJidlZFpghtgbD4I2VRHZvCwFFr/SrFM9pwrgbJNZ1MtyTSVgI5NlNb0aMfLR+/r6jJt3ROeY2yh2QQ2dx+YqgdbuL5ChFWOQU+ErSDnUq7yr+pjBQ==' },
    @{ Name = 'EM EV TLS CA - G3C (via emSign)'; Sha1 = 'C988F3D0CC1629FFC43165D6464C9B6EF73D6E48'; B64 = 'MIIDaDCCAu6gAwIBAgIQAIAJkopkEHYBrwX6LxnxCTAKBggqhkjOPQQDAzBWMQswCQYDVQQGEwJJTjElMCMGA1UECgwcZU11ZGhyYSBUZWNobm9sb2dpZXMgTGltaXRlZDEgMB4GA1UEAwwXZW1TaWduIFJvb3QgVExTIENBIC0gRzMwHhcNMjUwNDMwMTUzOTMxWhcNNDAwNDI4MTgzMDAwWjBAMQswCQYDVQQGEwJVUzEUMBIGA1UECgwLZU11ZGhyYSBJbmMxGzAZBgNVBAMMEkVNIEVWIFRMUyBDQSAtIEczQzB2MBAGByqGSM49AgEGBSuBBAAiA2IABC1W6Cmm+4LGSkaX2WCvlR6QZ66R+SbTQwbaSKiq63hptqZ+bSdCbgRCqgOzbhmI3/Y2UU7hzwf1uks/8qBmRa59MdWGatMao9JG6wsxd+6Zy3VmxRbOdICKZo+R8MR/yqOCAZUwggGRMB8GA1UdIwQYMBaAFOE6qyF88eNNVK7Ucrf9i1lCIJBUMB0GA1UdDgQWBBSmJV6AqU2385Z8G8eR35K87kvrXzAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwXAYDVR0gBFUwUzA5BgsrBgEEAYOOIQEAATAqMCgGCCsGAQUFBwIBFhxodHRwOi8vcmVwb3NpdG9yeS5lbXNpZ24uY29tMA0GCysGAQQBg44hAQJ4MAcGBWeBDAEBMHUGCCsGAQUFBwEBBGkwZzAkBggrBgEFBQcwAYYYaHR0cDovL29jc3AtYy5lbXNpZ24uY29tMD8GCCsGAQUFBzAChjNodHRwOi8vcmVwb3NpdG9yeS5lbXNpZ24uY29tL2NlcnRzL0VNRVZUTFNDQUczQy5wN2MwNwYDVR0fBDAwLjAsoCqgKIYmaHR0cDovL2NybC5lbXNpZ24uY29tLz9yb290dGxzY2FnMy5jcmwwDgYDVR0PAQH/BAQDAgEGMBIGA1UdEwEB/wQIMAYBAf8CAQAwCgYIKoZIzj0EAwMDaAAwZQIwBy1wTnzH4lXDTy2PD91ysLjvnlVN3JbJmc1BIMEU8TOLkYVHibMc9HeyonikFSfmAjEA6uE3QRxlrZTz/1/lXPTEMUhoauAtycagtlNFp79kjhvNa2/fAal1GNxsOutLnr0Y' },
    @{ Name = 'EM EV TLS CA - G3C (via Sectigo AAA)'; Sha1 = '8588708E875A1F8AA83524F3559080F7E17068C7'; B64 = 'MIIDzTCCArWgAwIBAgIQBfB4dS3s56G7hZXTAxZxcTANBgkqhkiG9w0BAQsFADB7MQswCQYDVQQGEwJHQjEbMBkGA1UECAwSR3JlYXRlciBNYW5jaGVzdGVyMRAwDgYDVQQHDAdTYWxmb3JkMRowGAYDVQQKDBFDb21vZG8gQ0EgTGltaXRlZDEhMB8GA1UEAwwYQUFBIENlcnRpZmljYXRlIFNlcnZpY2VzMB4XDTI1MDUxNDAwMDAwMFoXDTI4MTIzMTIzNTk1OVowQDELMAkGA1UEBhMCVVMxFDASBgNVBAoMC2VNdWRocmEgSW5jMRswGQYDVQQDDBJFTSBFViBUTFMgQ0EgLSBHM0MwdjAQBgcqhkjOPQIBBgUrgQQAIgNiAAQtVugppvuCxkpGl9lgr5UekGeukfkm00MG2kioqut4abamfm0nQm4EQqoDs24ZiN/2NlFO4c8H9bpLP/KgZkWufTHVhmrTGqPSRusLMXfumct1ZsUWznSAimaPkfDEf8qjggE0MIIBMDAfBgNVHSMEGDAWgBSgEQojPpbxB+zirynvgqV/0DCktDAdBgNVHQ4EFgQUpiVegKlNt/OWfBvHkd+SvO5L618wDgYDVR0PAQH/BAQDAgEGMBIGA1UdEwEB/wQIMAYBAf8CAQAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMDAGA1UdIAQpMCcwBwYFZ4EMAQEwDQYLKwYBBAGDjiEBAAEwDQYLKwYBBAGDjiEBAngwQwYDVR0fBDwwOjA4oDagNIYyaHR0cDovL2NybC5jb21vZG9jYS5jb20vQUFBQ2VydGlmaWNhdGVTZXJ2aWNlcy5jcmwwNAYIKwYBBQUHAQEEKDAmMCQGCCsGAQUFBzABhhhodHRwOi8vb2NzcC5jb21vZG9jYS5jb20wDQYJKoZIhvcNAQELBQADggEBAIFfHiaMSjnrdZygRYV4ekRBmG7ybcFk4hGQWMMyMFi0cAEk2+dGfz5bl8IaIbLqz+uVJzhOgVEX7WqIfgbWfFsfPYHYjPHBmy7SlZ3BwA86IQ6qF/etpX2/qL9A5yRNN3nVowUpaqV6H3aCpkAh3wR2ZNkvAbLDvPz3J0l+sEWjWv+ZISOV4R+hXDTdPXV01lKxnbwqEsdo/sE2QkCDKwFrZh/0uKjWspydJj+jSaDLTrGNx50Dgmn7VEcD1ZI5FXjknVUiO25YlmS1dtVZSBbyUKSjlrP/yDwPcZunoAsmCAAtCKsFLviLaymzvsKIeSSDGxpSySp2VCxompGDk8g=' },
    @{ Name = 'EM OV CS CA - G3B (via emSign)'; Sha1 = 'D33195A244046A3263FB5E0E83C6803A5B2E7BC3'; B64 = 'MIIDWzCCAuCgAwIBAgIPeBiTP4+FtGGaKzX7qD0KMAoGCCqGSM49BAMDMFUxCzAJBgNVBAYTAklOMSUwIwYDVQQKDBxlTXVkaHJhIFRlY2hub2xvZ2llcyBMaW1pdGVkMR8wHQYDVQQDDBZlbVNpZ24gUm9vdCBDUyBDQSAtIEczMB4XDTI1MDQzMDE2MDgyN1oXDTQwMDQyODE4MzAwMFowPjELMAkGA1UEBhMCTkwxEzARBgNVBAoMCmVNdWRocmEgQlYxGjAYBgNVBAMMEUVNIE9WIENTIENBIC0gRzNCMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEqrsPsGZqz88B+tUr/P1MK/LLPDSX3nOVDlgZth/mlQmHQcjEkUObNKBDHK/+W/m0wcjorgAB+fTcitgsWfNjnpbNbD1NsM7+wmw9YlF7N3YTcnQ/jordd7EAZPKr5eBQo4IBizCCAYcwHwYDVR0jBBgwFoAUaHQD+IXKjVRQVx0W4MjeRQ2rzS0wHQYDVR0OBBYEFHn66OzP2w11jaRY1p+27dx8Sd7mMBMGA1UdJQQMMAoGCCsGAQUFBwMDMF4GA1UdIARXMFUwOQYLKwYBBAGDjiEBAAEwKjAoBggrBgEFBQcCARYcaHR0cDovL3JlcG9zaXRvcnkuZW1zaWduLmNvbTAOBgwrBgEEAYOOIQECgUgwCAYGZ4EMAQQBMHQGCCsGAQUFBwEBBGgwZjAkBggrBgEFBQcwAYYYaHR0cDovL29jc3AtYi5lbXNpZ24uY29tMD4GCCsGAQUFBzAChjJodHRwOi8vcmVwb3NpdG9yeS5lbXNpZ24uY29tL2NlcnRzL0VNT1ZDU0NBRzNCLnA3YzA2BgNVHR8ELzAtMCugKaAnhiVodHRwOi8vY3JsLmVtc2lnbi5jb20vP3Jvb3Rjc2NhZzMuY3JsMA4GA1UdDwEB/wQEAwIBBjASBgNVHRMBAf8ECDAGAQH/AgEAMAoGCCqGSM49BAMDA2kAMGYCMQC/zr03YEELsNKqzUtH2dVFi4BaKFdyrr/6G1dIWp8KAVgsw2/u4lyyboUDaC9jDvICMQC9HTxku84tNX9dmHOtH7aC2Nu/JM0jTf6KjZAvovCJzzv1IFWSZpVLZibynMLXOrM=' },
    @{ Name = 'EM OV CS CA - G3B (via Sectigo AAA)'; Sha1 = 'EE26D7AE3CC71718F6E07D7D0095076F0C158F63'; B64 = 'MIIDwzCCAqugAwIBAgIQQeMrc6wYXWzg10gLrPhOaTANBgkqhkiG9w0BAQsFADB7MQswCQYDVQQGEwJHQjEbMBkGA1UECAwSR3JlYXRlciBNYW5jaGVzdGVyMRAwDgYDVQQHDAdTYWxmb3JkMRowGAYDVQQKDBFDb21vZG8gQ0EgTGltaXRlZDEhMB8GA1UEAwwYQUFBIENlcnRpZmljYXRlIFNlcnZpY2VzMB4XDTI1MDUxNDAwMDAwMFoXDTI4MTIzMTIzNTk1OVowPjELMAkGA1UEBhMCTkwxEzARBgNVBAoMCmVNdWRocmEgQlYxGjAYBgNVBAMMEUVNIE9WIENTIENBIC0gRzNCMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEqrsPsGZqz88B+tUr/P1MK/LLPDSX3nOVDlgZth/mlQmHQcjEkUObNKBDHK/+W/m0wcjorgAB+fTcitgsWfNjnpbNbD1NsM7+wmw9YlF7N3YTcnQ/jordd7EAZPKr5eBQo4IBLDCCASgwHwYDVR0jBBgwFoAUoBEKIz6W8Qfs4q8p74Klf9AwpLQwHQYDVR0OBBYEFHn66OzP2w11jaRY1p+27dx8Sd7mMA4GA1UdDwEB/wQEAwIBBjASBgNVHRMBAf8ECDAGAQH/AgEAMBMGA1UdJQQMMAoGCCsGAQUFBwMDMDIGA1UdIAQrMCkwCAYGZ4EMAQQBMA0GCysGAQQBg44hAQABMA4GDCsGAQQBg44hAQKBSDBDBgNVHR8EPDA6MDigNqA0hjJodHRwOi8vY3JsLmNvbW9kb2NhLmNvbS9BQUFDZXJ0aWZpY2F0ZVNlcnZpY2VzLmNybDA0BggrBgEFBQcBAQQoMCYwJAYIKwYBBQUHMAGGGGh0dHA6Ly9vY3NwLmNvbW9kb2NhLmNvbTANBgkqhkiG9w0BAQsFAAOCAQEAIh0OzWsdBYhY7+yxPGyosRL+uUSN6/Mrd30fBuougs5FVWQ3cRgLhImnUUgXADY9buugWhMmfT3cyU5T1MFXZXClCILm5mfg+oqm/ikUWJl3Ayf2GvYf2O++6vzqvkn418nI5Lpdt0O7Li524YS/3CZHDA+zZQ7rrHpZXgQ9Hmp8AUQ4FneaGjNanFoYxiZxwCyW5z4u7AoK0bmOVqKtcDFoVNz5++G9TYDJdhWOXmKKgzqRo1MK0PjazNT7MGHqknP93SqMVJ1gmX/qNWxYteAjlvFanQxBNX/+TRxDjuFsEHO/7BvqSpIIck0UEEfizhN8zNpWLbclEqVgl0g39A==' },
    @{ Name = 'EM OV CS CA - G3C (via emSign)'; Sha1 = '1F4A5D178F8B6A8EF0F6784155BD5FEA91158B12'; B64 = 'MIIDWzCCAuGgAwIBAgIPVEt/kiwxriIKSpIpFvbmMAoGCCqGSM49BAMDMFUxCzAJBgNVBAYTAklOMSUwIwYDVQQKDBxlTXVkaHJhIFRlY2hub2xvZ2llcyBMaW1pdGVkMR8wHQYDVQQDDBZlbVNpZ24gUm9vdCBDUyBDQSAtIEczMB4XDTI1MDQzMDE2Mzg0MFoXDTQwMDQyODE4MzAwMFowPzELMAkGA1UEBhMCVVMxFDASBgNVBAoMC2VNdWRocmEgSW5jMRowGAYDVQQDDBFFTSBPViBDUyBDQSAtIEczQzB2MBAGByqGSM49AgEGBSuBBAAiA2IABLnw4GNHAqacpqrdRDz9YsQbm6SyCSRpfXl052kaEm+5avs+NML9O8X0gwYfJl3CpYQ2NHMjUQQ2so8E84YjYwZ5gOJQPgJTLWVGI1pxZaJZoe9Jb3QbX97neHII5Lj676OCAYswggGHMB8GA1UdIwQYMBaAFGh0A/iFyo1UUFcdFuDI3kUNq80tMB0GA1UdDgQWBBTub3MpjW+/p7Mbw5U09hVWRHhmtzATBgNVHSUEDDAKBggrBgEFBQcDAzBeBgNVHSAEVzBVMDkGCysGAQQBg44hAQABMCowKAYIKwYBBQUHAgEWHGh0dHA6Ly9yZXBvc2l0b3J5LmVtc2lnbi5jb20wDgYMKwYBBAGDjiEBAoFIMAgGBmeBDAEEATB0BggrBgEFBQcBAQRoMGYwJAYIKwYBBQUHMAGGGGh0dHA6Ly9vY3NwLWMuZW1zaWduLmNvbTA+BggrBgEFBQcwAoYyaHR0cDovL3JlcG9zaXRvcnkuZW1zaWduLmNvbS9jZXJ0cy9FTU9WQ1NDQUczQy5wN2MwNgYDVR0fBC8wLTAroCmgJ4YlaHR0cDovL2NybC5lbXNpZ24uY29tLz9yb290Y3NjYWczLmNybDAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0TAQH/BAgwBgEB/wIBADAKBggqhkjOPQQDAwNoADBlAjEA8yPg8sntOhLwV2QXJ5LEaKSpo/fnE3uOgTadK73fDhyjiqr0Io9GVuQTYqBPbQ2TAjBgeuApmBS486+iei03lRrp139dPw6k28jjGNvWsAU7A1N86J1zPJVDEyVH+nMAWC0=' },
    @{ Name = 'EM OV CS CA - G3C (via Sectigo AAA)'; Sha1 = '7FDE64753A957B98F172FD2AB0D922551909C7BA'; B64 = 'MIIDxTCCAq2gAwIBAgIRANal72E7OQ8y75zfNJ93KEEwDQYJKoZIhvcNAQELBQAwezELMAkGA1UEBhMCR0IxGzAZBgNVBAgMEkdyZWF0ZXIgTWFuY2hlc3RlcjEQMA4GA1UEBwwHU2FsZm9yZDEaMBgGA1UECgwRQ29tb2RvIENBIExpbWl0ZWQxITAfBgNVBAMMGEFBQSBDZXJ0aWZpY2F0ZSBTZXJ2aWNlczAeFw0yNTA1MTQwMDAwMDBaFw0yODEyMzEyMzU5NTlaMD8xCzAJBgNVBAYTAlVTMRQwEgYDVQQKDAtlTXVkaHJhIEluYzEaMBgGA1UEAwwRRU0gT1YgQ1MgQ0EgLSBHM0MwdjAQBgcqhkjOPQIBBgUrgQQAIgNiAAS58OBjRwKmnKaq3UQ8/WLEG5uksgkkaX15dOdpGhJvuWr7PjTC/TvF9IMGHyZdwqWENjRzI1EENrKPBPOGI2MGeYDiUD4CUy1lRiNacWWiWaHvSW90G1/e53hyCOS4+u+jggEsMIIBKDAfBgNVHSMEGDAWgBSgEQojPpbxB+zirynvgqV/0DCktDAdBgNVHQ4EFgQU7m9zKY1vv6ezG8OVNPYVVkR4ZrcwDgYDVR0PAQH/BAQDAgEGMBIGA1UdEwEB/wQIMAYBAf8CAQAwEwYDVR0lBAwwCgYIKwYBBQUHAwMwMgYDVR0gBCswKTAIBgZngQwBBAEwDQYLKwYBBAGDjiEBAAEwDgYMKwYBBAGDjiEBAoFIMEMGA1UdHwQ8MDowOKA2oDSGMmh0dHA6Ly9jcmwuY29tb2RvY2EuY29tL0FBQUNlcnRpZmljYXRlU2VydmljZXMuY3JsMDQGCCsGAQUFBwEBBCgwJjAkBggrBgEFBQcwAYYYaHR0cDovL29jc3AuY29tb2RvY2EuY29tMA0GCSqGSIb3DQEBCwUAA4IBAQBgGbZ+NDkn2Fr5dCVs/51+6cT5iEPP1stglX4TI2UaNh9LmPdltxXw3crV8gK5L8i+d+ScGibFy0obtBHNTldXDgLnAHIzY3n+pceyKj7O2ae9CYnyuu5RBifeI3XYKowWE+oXnVGFOy0CvCJ/JsdP6aYgoCa9hGnAYicYQQL6MOMQZ5PuhcSM8t4giIzrpgrEkUP93GYEemDG5Z8BYwKs80lLeQNBnwg/VHJ3iEF/o84RXYwrlrKqJ+33C094o6LNikMJCqOK/WYHZSR8Xyhf8GXFAj5pZuI6Bcpug2V/E7i42j8OBhc25aNYDqK+B0qmb90sjNUBg3tJeSIb/599' }
)

# fleetd installs under the 64-bit Program Files even when this runs in 32-bit PowerShell.
$ProgramFiles64 = $env:ProgramW6432
if (-not $ProgramFiles64) { $ProgramFiles64 = $env:ProgramFiles }
if (-not $ProgramFiles64) { $ProgramFiles64 = 'C:\Program Files' }
$OrbitBundle = [IO.Path]::Combine($ProgramFiles64, 'Orbit', 'certs.pem')
$OrbitSvcKey = 'HKLM:\SYSTEM\CurrentControlSet\Services\Fleet osquery'
$script:ExitCode = 0

function Write-Log([string]$Message) { Write-Output $Message }
function Write-Warn([string]$Message) {
    Write-Output "WARN: $Message"
    if ($script:ExitCode -eq 0) { $script:ExitCode = 2 }
}

function New-CertFromB64([string]$B64) {
    return New-Object System.Security.Cryptography.X509Certificates.X509Certificate2 (, [Convert]::FromBase64String($B64))
}

function Add-DisallowedIntermediates {
    $store = New-Object System.Security.Cryptography.X509Certificates.X509Store('Disallowed', 'LocalMachine')
    $store.Open([System.Security.Cryptography.X509Certificates.OpenFlags]::ReadWrite)
    $added = 0
    try {
        $have = @{}
        foreach ($c in $store.Certificates) { $have[$c.Thumbprint] = $true }
        foreach ($i in $Intermediates) {
            $c = New-CertFromB64 $i.B64
            if ($c.Thumbprint -ne $i.Sha1) { throw "embedded certificate '$($i.Name)' failed its thumbprint check" }
            if (-not $have.ContainsKey($c.Thumbprint)) { $store.Add($c); $added++ }
        }
    } finally {
        $store.Close()
    }
    Write-Log "DISALLOWED: added $added, already present $($Intermediates.Count - $added) (of $($Intermediates.Count))"
}

# Count embedded intermediates that Windows now reports as explicitly distrusted.
function Test-DisallowedIntermediates {
    $distrusted = 0
    foreach ($i in $Intermediates) {
        $chain = New-Object System.Security.Cryptography.X509Certificates.X509Chain
        $chain.ChainPolicy.RevocationMode = [System.Security.Cryptography.X509Certificates.X509RevocationMode]::NoCheck
        [void]$chain.Build((New-CertFromB64 $i.B64))
        $flags = @($chain.ChainStatus | ForEach-Object { $_.Status.ToString() })
        if ($flags -contains 'ExplicitDistrust') { $distrusted++ }
        $chain.Reset()
    }
    return $distrusted
}

# Return the bundle text without the blocked roots. Comment lines above a removed
# certificate are dropped with it; everything else is kept byte for byte.
function Remove-BlockedFromPem([string]$Text) {
    $sb = New-Object System.Text.StringBuilder
    $endMarker = '-----END CERTIFICATE-----'
    $removed = 0
    $pos = 0
    while ($pos -lt $Text.Length) {
        $i = $Text.IndexOf($endMarker, $pos)
        if ($i -lt 0) { [void]$sb.Append($Text.Substring($pos)); break }
        $j = $Text.IndexOf("`n", $i)
        if ($j -lt 0) { $j = $Text.Length - 1 }
        $segment = $Text.Substring($pos, $j - $pos + 1)
        $pos = $j + 1
        $drop = $false
        $m = [regex]::Match($segment, '(?s)-----BEGIN CERTIFICATE-----(.*?)-----END CERTIFICATE-----')
        if ($m.Success) {
            try {
                $c = New-CertFromB64 ($m.Groups[1].Value -replace '\s', '')
                if ($BlockedRootSha1 -contains $c.Thumbprint) { $drop = $true }
            } catch { }
        }
        if ($drop) { $removed++ } else { [void]$sb.Append($segment) }
    }
    return @{ Text = $sb.ToString(); Removed = $removed }
}

function Get-PemThumbprints([string]$Text) {
    $set = @{}
    foreach ($m in [regex]::Matches($Text, '(?s)-----BEGIN CERTIFICATE-----(.*?)-----END CERTIFICATE-----')) {
        try { $set[(New-CertFromB64 ($m.Groups[1].Value -replace '\s', '')).Thumbprint] = $true } catch { }
    }
    return $set
}

# A CA's public key, independent of which copy (self-signed or cross-signed) carries it.
function Get-KeyId($Cert) {
    return $Cert.GetKeyAlgorithm() + ':' + $Cert.GetKeyAlgorithmParametersString() + ':' + $Cert.GetPublicKeyString()
}

# Key identifier (hex) from a certificate extension: Subject Key Identifier (2.5.29.14) is an
# OCTET STRING; Authority Key Identifier (2.5.29.35) is SEQUENCE { [0] keyIdentifier, ... }.
function Get-ExtKeyId($Cert, [string]$Oid) {
    $e = $Cert.Extensions | Where-Object { $_.Oid.Value -eq $Oid } | Select-Object -First 1
    if (-not $e) { return '' }
    $d = $e.RawData
    $i = 0
    if ($Oid -eq '2.5.29.35') {
        if ($d.Length -lt 4 -or $d[0] -ne 0x30) { return '' }
        $i = 2
        if ($d[1] -band 0x80) { $i = 2 + ($d[1] -band 0x7f) }
        if ($i + 1 -ge $d.Length -or $d[$i] -ne 0x80) { return '' }
    } elseif ($d.Length -lt 2 -or $d[0] -ne 0x04) {
        return ''
    }
    $len = [int]$d[$i + 1]
    $start = $i + 2
    if ($len -band 0x80) {
        $n = $len -band 0x7f
        $len = 0
        for ($j = 0; $j -lt $n; $j++) { $len = ($len * 256) + $d[$start + $j] }
        $start += $n
    }
    if ($start + $len -gt $d.Length) { return '' }
    return ([BitConverter]::ToString($d, $start, $len) -replace '-', '')
}

# Self-signed as OpenSSL decides it for a trust anchor: issuer equals subject, and the
# Authority Key Identifier is absent or equals the Subject Key Identifier. A same-name
# rollover certificate signed by an older key carries the old key's identifier and fails.
function Test-SelfSigned($Cert) {
    if ($Cert.Subject -ne $Cert.Issuer) { return $false }
    $aki = Get-ExtKeyId $Cert '2.5.29.35'
    $ski = Get-ExtKeyId $Cert '2.5.29.14'
    return (-not $aki -or -not $ski -or $aki -eq $ski)
}

# Trust anchors OpenSSL can end a chain on: the self-signed certificates in a PEM bundle,
# keyed by subject and public key. A bundle certificate that only shares a key with a
# chain CA, without being a self-signed root, cannot complete the chain.
function Get-PemAnchorIds([string]$Text) {
    $set = @{}
    foreach ($m in [regex]::Matches($Text, '(?s)-----BEGIN CERTIFICATE-----(.*?)-----END CERTIFICATE-----')) {
        try {
            $c = New-CertFromB64 ($m.Groups[1].Value -replace '\s', '')
            if (Test-SelfSigned $c) { $set[$c.Subject + '|' + (Get-KeyId $c)] = $true }
        } catch { }
    }
    return $set
}

function Get-OrbitEnv([string]$Name) {
    try {
        $envList = (Get-ItemProperty -Path $OrbitSvcKey -Name Environment -ErrorAction Stop).Environment
    } catch {
        return ''
    }
    foreach ($line in $envList) {
        if ($line -like "$Name=*") { return $line.Substring($Name.Length + 1).Trim('"') }
    }
    return ''
}

# fleetd packages built before Fleet 4.93.0 pass these settings as service arguments
# (ImagePath) instead of the Environment value.
function Get-OrbitArg([string]$Flag) {
    try {
        $imagePath = (Get-ItemProperty -Path $OrbitSvcKey -Name ImagePath -ErrorAction Stop).ImagePath
    } catch {
        return ''
    }
    $m = [regex]::Match([string]$imagePath, "--$Flag[= ]+(?:""([^""]+)""|(\S+))")
    if (-not $m.Success) { return '' }
    if ($m.Groups[1].Success) { return $m.Groups[1].Value }
    return $m.Groups[2].Value
}

function Get-OrbitSetting([string]$EnvName, [string]$Flag) {
    $v = Get-OrbitEnv $EnvName
    if (-not $v) { $v = Get-OrbitArg $Flag }
    return $v
}

# Certificates of the chain Windows builds for the Fleet server, leaf first, root last.
function Get-ServerChain([uri]$Uri) {
    $port = $Uri.Port
    if ($port -le 0) { $port = 443 }
    $tcp = New-Object System.Net.Sockets.TcpClient
    $iar = $tcp.BeginConnect($Uri.Host, $port, $null, $null)
    if (-not $iar.AsyncWaitHandle.WaitOne(15000)) { $tcp.Close(); throw "connect to $($Uri.Host):$port timed out" }
    $tcp.EndConnect($iar)
    # Bound the TLS handshake too: a server that accepts TCP but never answers would
    # otherwise block AuthenticateAsClient until Fleet's script timeout kills the run.
    $tcp.ReceiveTimeout = 15000
    $tcp.SendTimeout = 15000
    $script:ServerChain = @()
    $callback = [System.Net.Security.RemoteCertificateValidationCallback] {
        param($sender, $cert, $chain, $errors)
        # Copy each certificate so it stays valid after the chain is disposed.
        $script:ServerChain = @($chain.ChainElements | ForEach-Object { New-CertFromB64 ([Convert]::ToBase64String($_.Certificate.RawData)) })
        return $true
    }
    $ssl = New-Object System.Net.Security.SslStream($tcp.GetStream(), $false, $callback)
    try {
        $ssl.AuthenticateAsClient($Uri.Host, $null, [System.Security.Authentication.SslProtocols]::Tls12, $false)
    } finally {
        $ssl.Dispose()
        $tcp.Close()
    }
    return , $script:ServerChain
}

function Repair-OrbitBundle {
    if (-not (Test-Path -LiteralPath $OrbitBundle)) { Write-Log "FLEETD: $OrbitBundle not present (fleetd not installed)"; return }
    $text = [IO.File]::ReadAllText($OrbitBundle)
    $result = Remove-BlockedFromPem $text
    if ($result.Removed -eq 0) { Write-Log "FLEETD: $OrbitBundle already clean"; return }

    $kept = Get-PemThumbprints $result.Text
    if ($kept.Count -lt 50) { Write-Warn "FLEETD: filtered bundle failed sanity check; not changed"; return }
    foreach ($fp in $BlockedRootSha1) { if ($kept.ContainsKey($fp)) { Write-Warn "FLEETD: filter did not remove $fp; not changed"; return } }

    $pin = Get-OrbitSetting 'ORBIT_FLEET_CERTIFICATE' 'fleet-certificate'
    $url = Get-OrbitSetting 'ORBIT_FLEET_URL' 'fleet-url'
    if ($pin) {
        Write-Log "FLEETD: osquery pins $pin, so certs.pem is unused; cleaning it anyway"
    } elseif ($url) {
        try { $chain = Get-ServerChain ([uri]$url) } catch { Write-Warn "FLEETD: could not reach Fleet server ($($_.Exception.Message)); not changed"; return }
        if ($chain.Count -eq 0) { Write-Warn "FLEETD: no certificate chain from Fleet server; not changed"; return }
        $thumbs = @($chain | ForEach-Object { $_.Thumbprint })
        $bad = @($Intermediates | Where-Object { $thumbs -contains $_.Sha1 })
        if ($bad.Count -gt 0) { Write-Warn "FLEETD: Fleet server chains through a COMPROMISED intermediate ($($bad[0].Name)). Replace the server certificate now; not changed"; return }
        # Windows may extend the chain through a cross-signed copy of a root to an older root
        # that the bundle does not carry. osquery's OpenSSL stops at the first CA it finds as
        # a self-signed root in the bundle, so match each CA in the chain by subject and public
        # key against the bundle's self-signed roots, not on the thumbprint of the last certificate.
        $anchors = Get-PemAnchorIds $result.Text
        $anchor = @($chain | Select-Object -Skip 1 | Where-Object { $anchors.ContainsKey($_.Subject + '|' + (Get-KeyId $_)) }) | Select-Object -First 1
        if (-not $anchor) { Write-Warn "FLEETD: no CA in the Fleet server chain ($($thumbs -join ', ')) matches a self-signed root left in the filtered bundle; not changed"; return }
        Write-Log "FLEETD: Fleet server chain is anchored by '$($anchor.Subject)', a self-signed root that stays in the bundle"
    } else {
        Write-Warn "FLEETD: Fleet URL not found under $OrbitSvcKey (Environment or ImagePath); cannot prove safety; not changed"; return
    }

    $backup = "$OrbitBundle.pre-emudhra"
    if (-not (Test-Path -LiteralPath $backup)) { Copy-Item -LiteralPath $OrbitBundle -Destination $backup }
    $tmp = "$OrbitBundle.new"
    [IO.File]::WriteAllText($tmp, $result.Text, (New-Object System.Text.UTF8Encoding $false))
    [IO.File]::Replace($tmp, $OrbitBundle, [NullString]::Value)
    Write-Log "FLEETD: removed $($result.Removed) root(s) from $OrbitBundle (backup: $backup). osquery reloads it on its next connection."
}

function Test-AutoUpdatePolicy {
    $key = 'HKLM:\SOFTWARE\Policies\Microsoft\SystemCertificates\AuthRoot'
    $p = Get-ItemProperty -Path $key -ErrorAction SilentlyContinue
    if ($p -and ($p.PSObject.Properties.Name -contains 'DisableRootAutoUpdate') -and $p.DisableRootAutoUpdate -eq 1) {
        Write-Log "POLICY: DisableRootAutoUpdate=1. Microsoft's own block cannot arrive; the Disallowed entries above cover this incident."
    }
    if ($p -and ($p.PSObject.Properties.Name -contains 'EnableDisallowedCertAutoUpdate') -and $p.EnableDisallowedCertAutoUpdate -eq 0) {
        Write-Log "POLICY: EnableDisallowedCertAutoUpdate=0. Future Microsoft distrust updates will not arrive automatically."
    }
}

function Invoke-Main {
    try {
        Write-Log '----- Windows certificate store -----'
        Add-DisallowedIntermediates
        Test-AutoUpdatePolicy
        Write-Log '----- fleetd osquery CA bundle -----'
        Repair-OrbitBundle
        Write-Log '----- verification -----'
        $n = Test-DisallowedIntermediates
        if ($n -eq $Intermediates.Count) { Write-Log "VERIFY: $n of $($Intermediates.Count) compromised intermediates report ExplicitDistrust" }
        else { Write-Warn "VERIFY: only $n of $($Intermediates.Count) compromised intermediates report ExplicitDistrust" }
        if (Test-Path -LiteralPath $OrbitBundle) {
            $left = Get-PemThumbprints ([IO.File]::ReadAllText($OrbitBundle))
            if (@($BlockedRootSha1 | Where-Object { $left.ContainsKey($_) }).Count -eq 0) { Write-Log "VERIFY: $OrbitBundle PROTECTED" }
            else { Write-Warn "VERIFY: $OrbitBundle still trusts a compromised root" }
        }
        Write-Log 'Node.js and pip-installed certifi carry their own CA copy: update them separately.'
    } catch {
        Write-Output "ERROR: $($_.Exception.Message)"
        exit 1
    }
    exit $script:ExitCode
}

if ($MyInvocation.InvocationName -ne '.') { Invoke-Main }
