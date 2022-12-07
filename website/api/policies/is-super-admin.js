/**
 * is-super-admin
 *
 * A simple policy that blocks requests from non-super-admins.
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
      return res.redirect('/customers/login?admin');
    }
  }//•

  // Then check that this user is a "super admin".
  if (!req.me.isSuperAdmin) {
    return res.forbidden();
  }//•

  // IWMIH, we've got ourselves a "super admin".
  return proceed();

};
