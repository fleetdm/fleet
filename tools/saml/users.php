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
        // sso_user_4_team_maintainer has FLEET_JIT_USER_ROLE_TEAM_1 attribute to be added as maintainer
        // of team with ID 1, its login will fail if team with ID 1 doesn't exist.
        'sso_user_4_team_maintainer:user123#' => array(
            'uid' => array('4'),
            'eduPersonAffiliation' => array('group1'),
            'http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name' => array('SSO User 4'),
            'email' => 'sso_user_4_team_maintainer@example.com',
            'FLEET_JIT_USER_ROLE_TEAM_1' => 'maintainer',
        ),
        // sso_user_5_team_admin has FLEET_JIT_USER_ROLE_TEAM_1 attribute to be added as admin
        // of team with ID 1, its login will fail if team with ID 1 doesn't exist.
        // It also sets FLEET_JIT_USER_ROLE_GLOBAL and FLEET_JIT_USER_ROLE_TEAM_2 to `null` which means
        // Fleet will ignore such fields.
        'sso_user_5_team_admin:user123#' => array(
            'uid' => array('5'),
            'eduPersonAffiliation' => array('group1'),
            'http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name' => array('SSO User 5'),
            'email' => 'sso_user_5_team_admin@example.com',
            'FLEET_JIT_USER_ROLE_TEAM_1' => 'admin',
            'FLEET_JIT_USER_ROLE_GLOBAL' => 'null',
            'FLEET_JIT_USER_ROLE_TEAM_2' => 'null',
        ),
        // sso_user_6_global_observer has all FLEET_JIT_USER_ROLE_* attributes set to null, so it
        // will be added as global observer (default).
        'sso_user_6_global_observer:user123#' => array(
            'uid' => array('6'),
            'eduPersonAffiliation' => array('group1'),
            'http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name' => array('SSO User 6'),
            'email' => 'sso_user_6_global_observer@example.com',
            'FLEET_JIT_USER_ROLE_GLOBAL' => 'null',
            'FLEET_JIT_USER_ROLE_TEAM_1' => 'null',
        ),
        // sso_user_no_displayname does not have a displayName/fullName
        'sso_user_no_displayname:user123#' => array(
            'uid' => array('7'),
            'eduPersonAffiliation' => array('group1'),
            'email' => 'sso_user_no_displayname@example.com',
        ),
    ),

);
