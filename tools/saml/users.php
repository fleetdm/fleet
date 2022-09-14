<?php

$config = array(

    'admin' => array(
        'core:AdminPassword',
    ),

    'example-userpass' => array(
        'exampleauth:UserPass',
        // username: sso_user
        // password: user123#
        'sso_user:user123#' => array(
            'uid' => array('1'),
            'eduPersonAffiliation' => array('group1'),
            'displayname' => array('SSO User 1'),
            'email' => 'sso_user@example.com',
        ),
        'sso_user2:user123#' => array(
            'uid' => array('2'),
            'eduPersonAffiliation' => array('group1'),
            'http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name' => array('SSO User 2'),
            'email' => 'sso_user2@example.com',
        ),
    ),

);
