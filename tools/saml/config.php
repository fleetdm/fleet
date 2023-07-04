<?php

$metadata['https://localhost:8080'] = array(
    'AssertionConsumerService' => [
        'https://localhost:8080/api/v1/fleet/sso/callback',
        'https://localhost:8080/api/v1/fleet/mdm/sso/callback',
    ],
    'NameIDFormat' => 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddres',
    'simplesaml.nameidattribute' => 'email',
);

