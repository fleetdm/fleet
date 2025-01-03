/**
 * Module dependencies
 */
let { ConfidentialClientApplication } = require('@azure/msal-node');
let jwt = require('jsonwebtoken');

/**
 * Entra SSO Hook
 */
module.exports = function (sails) {

  let entraSSOClient;
  return {
    defaults: {
      entraSSO: {
        userModelIdentity: 'user',
      },
    },

    initialize: function (cb) {
      if (!sails.config.custom.entraClientId) {
        return cb();
      }

      sails.log('Entra SSO enabled. The built-in authorization mechanism will be disabled.');

      // Throw errors if required config variables are missing.
      if(!sails.config.custom.entraTenantId){
        throw new Error(`Missing config! No sails.config.custom.entraTenantId was configured. To replace this app's built-in authorization mechanism with Entra SSO, an entraTenantId value is required.`);
      }
      if(!sails.config.custom.entraClientSecret){
        throw new Error(`Missing config! No sails.config.custom.entraClientSecret was configured. To replace this app's built-in authorization mechanism with Entra SSO, an entraClientSecret value is required.`);
      }

      // [?]: https://learn.microsoft.com/en-us/entra/external-id/customers/tutorial-web-app-node-sign-in-sign-out#create-msal-configuration-object
      // Configure the SSO client application.
      entraSSOClient = new ConfidentialClientApplication({
        auth: {
          clientId: sails.config.custom.entraClientId,
          authority: `https://login.microsoftonline.com/${sails.config.custom.entraTenantId}`,
          clientSecret: sails.config.custom.entraClientSecret,
        },
      });

      var err;
      // Validate `userModelIdentity` config
      if (typeof sails.config.entraSSO.userModelIdentity !== 'string') {
        sails.config.entraSSO.userModelIdentity = 'user';
      }
      sails.config.entraSSO.userModelIdentity = sails.config.entraSSO.userModelIdentity.toLowerCase();
      // We must wait for the `orm` hook before acquiring our user model from `sails.models`
      // because it might not be ready yet.
      if (!sails.hooks.orm) {
        err = new Error();
        err.code = 'E_HOOK_INITIALIZE';
        err.name = 'Entra SSO Hook Error';
        err.message = 'The "Entra SSO" hook depends on the "orm" hook- cannot load the "Entra SSO" hook without it!';
        return cb(err);
      }
      sails.after('hook:orm:loaded', ()=>{

        // Look up configured user model
        var UserModel = sails.models[sails.config.entraSSO.userModelIdentity];

        if (!UserModel) {
          err = new Error();
          err.code = 'E_HOOK_INITIALIZE';
          err.name = 'Entra SSO Hook Error';
          err.message = 'Could not load the Entra SSO hook because `sails.config.passport.userModelIdentity` refers to an unknown model: "'+sails.config.entraSSO.userModelIdentity+'".';
          if (sails.config.entraSSO.userModelIdentity === 'user') {
            err.message += '\nThis option defaults to `user` if unspecified or invalid- maybe you need to set or correct it?';
          }
          return cb(err);
        }
        cb();
      });
    },

    routes: {
      before: {
        '/login': async (req, res, next) => {
          if (!sails.config.custom.entraClientId) {
            return next();
          }
          // Get the sso login url and redirect the user
          // [?]: https://learn.microsoft.com/en-us/javascript/api/%40azure/msal-node/confidentialclientapplication?view=msal-js-latest#@azure-msal-node-confidentialclientapplication-getauthcodeurl
          let entraAuthorizationUrl = await entraSSOClient.getAuthCodeUrl({
            redirectUri: `${sails.config.custom.baseUrl}/authorization-code/callback`,
            scopes: ['openid', 'profile', 'email', 'User.Read'],
          });
          // Redirect the user to the SSO login url.
          res.redirect(entraAuthorizationUrl);
        },
        '/authorization-code/callback': async (req, res, next) => {
          if (!sails.config.custom.entraClientId) {
            return next();
          }
          // Make sure there is a code query string.
          let codeToGetToken = req.query.code;
          if(!codeToGetToken){
            res.unauthorized();
          }
          // [?]: https://learn.microsoft.com/en-us/javascript/api/%40azure/msal-node/confidentialclientapplication?view=msal-js-latest#@azure-msal-node-confidentialclientapplication-acquiretokenbycode
          let responseFromEntra = await entraSSOClient.acquireTokenByCode({
            code: codeToGetToken,
            redirectUri: `${sails.config.custom.baseUrl}/authorization-code/callback`,
            scopes: ['openid', 'profile', 'email', 'User.Read'],
          });
          // Decode the accessToken in the response from Entra.
          let decodedToken = jwt.decode(responseFromEntra.accessToken);
          // Set the decoded token as the user's ssoUserInformation in their session.
          req.session.ssoUserInformation = decodedToken;
          // Redirect the user to the signup-sso-user-or-redirect endpoint.
          res.redirect('/entrance/signup-sso-user-or-redirect'); // Note: This action handles signing in/up users who authenticate through Microsoft Entra.
        },
        '/logout': async(req, res, next)=>{
          if (!sails.config.custom.entraClientId) {
            return next();
          }
          let logoutUri = `https://login.microsoftonline.com/${sails.config.custom.entraTenantId}/oauth2/v2.0/logout?post_logout_redirect_uri=${sails.config.custom.baseUrl}/`;
          delete req.session.userId;
          res.redirect(logoutUri);
        },
      },
    },
  };
};
