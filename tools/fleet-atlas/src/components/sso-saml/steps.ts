// The SAML SSO login flow for a Fleet admin, in the correct order.
// The quiz shuffles these at render time; user drags to reorder, then submits.

export type SSOStep = {
  key: string;
  // Who/what acts in this step. Used as a short lane label to help the user
  // visualize who's doing what.
  actor: "Admin" | "Fleet" | "IdP";
  text: string;
};

export const SSO_SAML_STEPS: SSOStep[] = [
  {
    key: "click-login",
    actor: "Admin",
    text: 'User clicks "Log in with SSO."',
  },
  {
    key: "fleet-redirect",
    actor: "Fleet",
    text: "Fleet responds with a 302 to the IdP with a signed SAML AuthnRequest.",
  },
  {
    key: "idp-auth",
    actor: "IdP",
    text: "IdP authenticates the user (whatever MFA).",
  },
  {
    key: "idp-post-acs",
    actor: "IdP",
    text: "IdP POSTs a signed SAML Response to Fleet's Assertion Consumer Service (ACS) endpoint.",
  },
  {
    key: "verify-sig",
    actor: "Fleet",
    text: "Fleet verifies the XML signature against the configured IdP cert.",
  },
  {
    key: "extract-claims",
    actor: "Fleet",
    text: "Extracts email + attribute claims; finds the user (or auto-provisions one if JIT is enabled).",
  },
  {
    key: "mint-session",
    actor: "Fleet",
    text: "Mints a session the same way local-password login does.",
  },
];
