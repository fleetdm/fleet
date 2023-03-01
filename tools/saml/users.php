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
        // sso_user_3_global_admin has FLEET_JIT_USER_ROLE_GLOBAL attribute to be added as global admin.
        'sso_user_3_global_admin:user123#' => array(
            'uid' => array('3'),
            'eduPersonAffiliation' => array('group1'),
            'http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name' => array('SSO User 3'),
            'email' => 'sso_user_3_global_admin@example.com',
            'FLEET_JIT_USER_ROLE_GLOBAL' => 'admin',
        ),
        // sso_team_user_3 has FLEET_JIT_USER_ROLE_TEAM_1 attribute to be added as maintainer
        // of team with ID 1, its login will fail if team with ID 1 doesn't exist.
        'sso_user_4_team_maintainer:user123#' => array(
            'uid' => array('4'),
            'eduPersonAffiliation' => array('group1'),
            'http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name' => array('SSO User 4'),
            'email' => 'sso_user_4_team_maintainer@example.com',
            'FLEET_JIT_USER_ROLE_TEAM_1' => 'maintainer',
        ),
    ),

);
