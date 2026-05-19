## Android agent 1.5.0 (May 14, 2026)

* Fixed certificate serial number truncation by sending serials to Fleet as hexadecimal.
* Made the agent retry DNS resolution failures during certificate enrollment when waking from Doze mode, and defer remaining certificates in the batch to the next cycle when DNS still fails.
* Made the agent always send the `platform` field on enrollment so the device is registered as Android.

## Android agent 1.4.0 (Apr 07, 2026)

* Made the certificate list scrollable
* Fixed background DNS resolution failures.
* Stopped polling certificates when the server reported them as permanently failed.
* Marked non-retryable SCEP failures (e.g. server rejection) as failed immediately instead of retrying 3 times.
* Fixed duplicate FAILED status reports.
* Made the agent treat HTTP 404 responses on certificate status updates as a signal that the template had been deleted server-side.
* Made enrollment failure messages include SCEP failInfo details instead of a generic error.
* Made certificate enrollment wait for CERT_INSTALL delegation to be available, preventing permanent failures after fresh MDM enrollment.
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
