/**
 * openStripeCheckout()
 *
 * Open the Stripe Checkout modal dialog and resolve when it is closed.
 *
 * -----------------------------------------------------------------
 * @param {String} stripePublishableKey
 * @param {String} billingEmailAddress
 * @param {String} headingText (optional)
 * @param {String} descriptionText (optional)
 * @param {String} buttonText (optional)
 * -----------------------------------------------------------------
 * @returns {Dictionary?}  (or undefined if the form was cancelled)
 *          e.g.
 *          {
 *            stripeToken: '…',
 *            billingCardLast4: '…',
 *            billingCardBrand: '…',
 *            billingCardExpMonth: '…',
 *            billingCardExpYear: '…'
 *          }
 * -----------------------------------------------------------------
 * Example usage:
 * ```
 * var billingInfo = await openStripeCheckout(
 *   'pk_test_Qz5RfDmVV5IunTFAHtDqDWn4',
 *   'foo@example.com'
 * );
 * ```
 */

parasails.registerUtility('openStripeCheckout', async function openStripeCheckout(stripePublishableKey, billingEmailAddress, headingText, descriptionText, buttonText) {

  // Cache (& use cached) "checkout handler" globally on the page so that we
  // don't end up configuring it more than once (i.e. so Stripe.js doesn't
  // complain).
  var CACHE_KEY = '_cachedStripeCheckoutHandler';
  if (!window[CACHE_KEY]) {
    window[CACHE_KEY] = StripeCheckout.configure({
      key: stripePublishableKey,
    });
  }
  var checkoutHandler = window[CACHE_KEY];

  // Track whether the "token" callback was triggered.
  // (If it has NOT at the time the "closed" callback is triggered, then we
  // know the checkout form was cancelled.)
  var hasTriggeredTokenCallback;

  // Build a Promise & send it back as our "thenable" (AsyncFunction's return value).
  // (this is necessary b/c we're wrapping an api that isn't `await`-compatible)
  return new Promise((resolve, reject)=>{
    try {
      // Open Stripe checkout.
      // (https://stripe.com/docs/checkout#integration-custom)
      checkoutHandler.open({
        name: headingText || 'NEW_APP_NAME',
        description: descriptionText || 'Link your credit card.',
        panelLabel: buttonText || 'Save card',
        email: billingEmailAddress,//« So that Stripe doesn't prompt for an email address
        locale: 'auto',
        zipCode: false,
        allowRememberMe: false,
        closed: ()=>{
          // If the Checkout dialog was cancelled, resolve undefined.
          if (!hasTriggeredTokenCallback) {
            resolve();
          }
        },
        token: (stripeData)=>{

          // After payment info has been successfully added, and a token
          // was obtained...
          hasTriggeredTokenCallback = true;

          // Normalize token and billing card info from Stripe and resolve
          // with that.
          let stripeToken = stripeData.id;
          let billingCardLast4 = stripeData.card.last4;
          let billingCardBrand = stripeData.card.brand;
          let billingCardExpMonth = String(stripeData.card.exp_month);
          let billingCardExpYear = String(stripeData.card.exp_year);

          resolve({
            stripeToken,
            billingCardLast4,
            billingCardBrand,
            billingCardExpMonth,
            billingCardExpYear
          });
        }//Œ
      });//_∏_
    } catch (err) {
      reject(err);
    }
  });//_∏_

});
