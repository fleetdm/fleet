# Device identity in Smallstep SCEP challenge requests

When Fleet prepares an Apple configuration profile containing `$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_<CA_NAME>`, it requests a challenge from the configured Smallstep challenge URL using the integration's HTTP Basic authentication credentials. The request includes the intended profile recipient:

```json
{
  "webhook": {
    "id": 1,
    "webhookEvent": "SCEPChallenge",
    "eventTimestamp": 1780000000,
    "name": "SCEPChallenge"
  },
  "event": {
    "scepServerUrl": "https://ca.example/scep/wifi",
    "payloadIdentifier": "random-request-uuid",
    "payloadTypes": ["com.apple.security.scep"],
    "device": {
      "uuid": "fleet-host-uuid",
      "serialNumber": "available-hardware-serial"
    }
  }
}
```

`event.device.uuid` is the host UUID associated with the profile recipient, matching `$FLEET_VAR_HOST_UUID`. It is an opaque Fleet inventory identity, not necessarily a hardware UUID. For user-channel delivery, Fleet resolves the enrollment ID to the host before requesting the challenge. `serialNumber` comes from Fleet inventory and is omitted when unavailable, including privacy-preserving enrollment modes. Neither value comes from a client-supplied certificate request.

Challenge services should authenticate Fleet's request and bind the returned challenge to the permitted certificate identity and intended CA/provisioner. The CA must enforce that binding against the requested certificate identity. Merely generating a unique password does not prevent a device from requesting another device's identity. This context identifies the intended recipient; it is not hardware attestation.

When testing the integration configuration, Fleet sends a probe with `event.device` omitted because no device is involved. An identity-binding service can return a non-enrollment probe response for that check; it must not issue usable unbound enrollment credentials. Existing challenge request fields and response handling are unchanged.
