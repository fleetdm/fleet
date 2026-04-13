## Android agent 1.4.0 (Apr 07, 2026)

* Make certificate list scrollable
* Fixed background DNS resolution failures.
* Stop polling certificates when the server reports them as permanently failed.
* Non-retryable SCEP failures (e.g. server rejection) now immediately mark the certificate as failed instead of retrying 3 times.
* Fixed duplicate FAILED status reports.
* Treat HTTP 404 on certificate status updates as a signal that the template has been deleted server-side.
* Include SCEP `failInfo` details in enrollment failure messages instead of a generic error.
* Wait for CERT_INSTALL delegation to be available before attempting certificate enrollment, preventing permanent failures after fresh MDM enrollment.
* Improved certificate installation failure messages to include delegation status and certificate alias.

## Android agent 1.3.0 (Feb 27, 2026)

* Improved debug screen, including adding last error message and logs.

## Android agent 1.2.0 (Feb 13, 2026)

* Fixed issue where certification installations incorrectly show failed statuses.
* Fixed issue where agent does not re-enroll after the host is deleted in Fleet.

## Android agent 1.1.0 (Jan 12, 2026)

* Automatically renew SCEP certs. Requires Fleet server v4.80 or higher.

## Android agent 1.0.2 (Jan 6, 2026)

* First release, supporting installing and removing certs with custom SCEP CAs.
