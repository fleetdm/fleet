<?php

$metadata['https://jve-fleetdm-snicket.ngrok.app'] = array(
    'AssertionConsumerService' => [
        'https://jve-fleetdm-snicket.ngrok.app/api/v1/fleet/sso/callback',
        'https://jve-fleetdm-snicket.ngrok.app/api/v1/fleet/mdm/sso/callback',
    ],
    'NameIDFormat' => 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddres',
    'simplesaml.nameidattribute' => 'email',
);

# used in integration tests and to validate SSO flows that use a
# separate application for MDM SSO (with a single
# AssertionConsumerService)
$metadata['mdm.test.com'] = array(
    'AssertionConsumerService' => [
        'https://jve-fleetdm-snicket.ngrok.app/api/v1/fleet/mdm/sso/callback',
    ],
    'NameIDFormat' => 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddres',
    'simplesaml.nameidattribute' => 'email',
);
