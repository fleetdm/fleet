<?php

$metadata['https://localhost:8080'] = array(
    'AssertionConsumerService' => [
        'https://localhost:8080/api/v1/fleet/sso/callback',
        'https://localhost:8080/api/v1/fleet/mdm/sso/callback',

    ],
    'NameIDFormat' => 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddres',
    'simplesaml.nameidattribute' => 'email',
);

$metadata['https://www.okta.com/saml2/service-provider/spxarorktvzekztynake'] = array(
    'AssertionConsumerService' => [
        'https://trial-1930681.okta.com/sso/saml2/0oabi74v9iGQEsLVT697',

    ],
    'NameIDFormat' => 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress',
    'simplesaml.nameidattribute' => 'email',
);

