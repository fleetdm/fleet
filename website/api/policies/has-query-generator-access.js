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

  // First, check whether the request comes from a logged-in user.
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

  // Check if this user can access the query generator.
  if (!req.me.canUseQueryGenerator) {
    return res.forbidden();
  // Then check that this user is a "super admin".
  } else if (!req.me.isSuperAdmin) {
    return res.forbidden();
  }

  // IWMIH, this user can access the query generator.
  return proceed();

};
