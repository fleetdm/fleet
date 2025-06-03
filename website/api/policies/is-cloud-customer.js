/**
 * is-cloud-customer
 *
 * A simple policy that allows requests to microsoft proxy endpoints from a cloud customer.
 *
 * For more about how to use policies, see:
 *   https://sailsjs.com/config/policies
 *   https://sailsjs.com/docs/concepts/policies
 *   https://sailsjs.com/docs/concepts/policies/access-control-and-permissions
 */
module.exports = async function (req, res, proceed) {

  // If an MS API KEY header was provided, check to see if it matches the entraSharedSecret.
  if (req.get('MS-API-KEY')) {
    if(req.get('MS-API-KEY') === sails.config.custom.cloudCustomerCompliancePartnerSharedSecret){
      return proceed();
    }
  }

  //--•
  // Otherwise, this request did not come from a cloud customer.
  return res.unauthorized();

};
