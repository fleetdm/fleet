/**
 * is-authorized-by-bearer-token
 *
 * A simple policy that allows any request from an authenticated user.
 *
 * For more about how to use policies, see:
 *   https://sailsjs.com/config/policies
 *   https://sailsjs.com/docs/concepts/policies
 *   https://sailsjs.com/docs/concepts/policies/access-control-and-permissions
 */
module.exports = async function (req, res, proceed) {

  // If an `Authorization` header is set, then we'll check the value to see if it matches one of the existing User records.
  if (req.get('authorization')) {
    let authorizationHeader = req.get('authorization');
    if(!_.startsWith(authorizationHeader, 'Bearer ')) {
      return res.unauthorized();
    }
    let tokenInAuthorizationHeader = authorizationHeader.split('Bearer ')[1];
    if(!tokenInAuthorizationHeader) {
      return res.unauthorized();
    }
    let userThisTokenBelongsTo = await User.findOne({apiToken: tokenInAuthorizationHeader});
    if(!userThisTokenBelongsTo){
      return res.unauthorized();
    } else {
      return proceed();
    }
  }

  //--•
  // Otherwise, this request did not come from a logged-in user.
  return res.unauthorized();

};
