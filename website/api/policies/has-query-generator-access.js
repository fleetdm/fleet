// FUTURE: remove this policy once the query generator is public
/**
 * has-query-generator-access
 *
 * A simple policy that blocks requests from users who have not been granted access to the query generator.
 *
 * For more about how to use policies, see:
 *   https://sailsjs.com/config/policies
 *   https://sailsjs.com/docs/concepts/policies
 *   https://sailsjs.com/docs/concepts/policies/access-control-and-permissions
 */
module.exports = async function (req, res, proceed) {

  // First, check whether the query generator is open to the public by checking the enablePublicQueryGenerator config value.
  // This is here to allow us to open the query generator to everyone after we internally QA it on the live website, without needing to redeploy the website.
  if(sails.config.custom.enablePublicQueryGenerator){
    return proceed();
  }
  // Then, check whether the request comes from a logged-in user.
  // > For more about where `req.me` comes from, check out this app's
  // > custom hook (`api/hooks/custom/index.js`).
  if (!req.me) {
    // Rather than use the standard res.unauthorized(), if the request did not come from a logged-in user,
    // we'll redirect them to an generic version of the customer login page.
    if (req.wantsJSON) {
      return res.sendStatus(401);
    } else {
      return res.redirect('/login');
    }
  }//•

  // Check that this user is a "super admin".
  if(!req.me.isSuperAdmin) {
    // Then check if this user can access the query generator.
    if (!req.me.canUseQueryGenerator) {
      return res.forbidden();
    }
  }

  // IWMIH, this user can access the query generator.
  return proceed();

};
