<?php

$metadata['https://localhost:8080'] = array(
    'AssertionConsumerService' => [
        'https://localhost:8080/api/v1/fleet/sso/callback',
    ],
    'NameIDFormat' => 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddres',
    'simplesaml.nameidattribute' => 'email',
);

# For MDM SSO SAML behind ngrok
$metadata['https://mnafleet.ngrok.app'] = array(
    'AssertionConsumerService' => [
        'https://mnafleet.ngrok.app/api/v1/fleet/mdm/sso/callback',
    ],
    'NameIDFormat' => 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddres',
    'simplesaml.nameidattribute' => 'email',
);

# Used in integration tests and to validate SSO flows that use a
# separate application for MDM SSO (with a single
# AssertionConsumerService)
$metadata['mdm.test.com'] = array(
    'AssertionConsumerService' => [
        'https://localhost:8080/api/v1/fleet/mdm/sso/callback',
    ],
    'NameIDFormat' => 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddres',
    'simplesaml.nameidattribute' => 'email',
);

# Used for testing when sso_settings.entity_id ("sso.test.com") is different than
# server_settings.server_url (usually "https://localhost:8080").
$metadata['sso.test.com'] = array(
    'AssertionConsumerService' => [
        'https://localhost:8080/api/v1/fleet/sso/callback',
    ],
    'NameIDFormat' => 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddres',
    'simplesaml.nameidattribute' => 'email',
);

# Used for testing when entity_id is not set, so that it matches the hostname (localhost).
$metadata['localhost'] = array(
    'AssertionConsumerService' => [
        'https://localhost:8080/api/v1/fleet/sso/callback',
    ],
    'NameIDFormat' => 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddres',
    'simplesaml.nameidattribute' => 'email',
);
