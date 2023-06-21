import Table from "./Table";

export default class TablePrivacyPreferences extends Table {
  // expose properties available from the chrome.privacy API as a virtual osquery "table"
  // https://developer.chrome.com/docs/extensions/reference/privacy/

  name = "privacy_preferences";
  propertyAPIs = {
    // all of type `types.ChromeSetting<boolean>` with default `true` unless otherwise specified

    // though all of these properties are documented, some have not been added to the
    // typings we are using: https://www.npmjs.com/package/@types/chrome and must be `@ts-ignore`d

    // network
    network_prediction_enabled: chrome.privacy.network.networkPredictionEnabled,
    // type `types.ChromeSetting<IPHandlingPolicy>`, default "default"
    // IPHandlingPolicy:
    // "default" | "default_public_and_private_interfaces" |
    // "default_public_interface_only" | "disable_non_proxied_udp"
    web_rtc_ip_handling_policy: chrome.privacy.network.webRTCIPHandlingPolicy,

    // services
    autofill_address_enabled: chrome.privacy.services.autofillAddressEnabled,
    autofill_credit_card_enabled:
      chrome.privacy.services.autofillCreditCardEnabled,
    // DEPRECATED and replaced with above two properties
    autofill_enabled: chrome.privacy.services.autofillEnabled,
    save_passwords_enabled: chrome.privacy.services.passwordSavingEnabled,
    safe_browsing_enabled: chrome.privacy.services.safeBrowsingEnabled,
    // default false
    safe_browsing_extended_reporting_enabled:
      chrome.privacy.services.safeBrowsingExtendedReportingEnabled,
    search_suggest_enabled: chrome.privacy.services.searchSuggestEnabled,
    // default false
    spelling_service_enabled: chrome.privacy.services.spellingServiceEnabled,
    translation_service_enabled:
      chrome.privacy.services.translationServiceEnabled,

    // websites
    // @ts-ignore
    ad_measurement_enabled: chrome.privacy.websites.adMeasurementEnabled,
    // default false
    do_not_track_enabled: chrome.privacy.websites.doNotTrackEnabled,
    // @ts-ignore
    fledge_enabled: chrome.privacy.websites.fledgeEnabled,
    hyperlink_auditing_enabled:
      chrome.privacy.websites.hyperlinkAuditingEnabled,
    // @ts-ignore, DEPRECATED
    privacy_sandbox_enabled: chrome.privacy.websites.privacySandboxEnabled,
    // WINDOWS AND CHROMEOS ONLY - if desired, can check the platform via `chrome.runtime.getPlatformInfo`
    // protected_content_enabled: chrome.privacy.websites.protectedContentEnabled,
    referrers_enabled: chrome.privacy.websites.referrersEnabled,
    third_party_cookies_allowed:
      chrome.privacy.websites.thirdPartyCookiesAllowed,
    // @ts-ignore
    topics_enabled: chrome.privacy.websites.topicsEnabled,
  };

  columns = Object.keys(this.propertyAPIs);

  async generate() {
    const result = {};

    // for (const property of this.columns) {
    for (const [property, propertyAPI] of Object.entries(this.propertyAPIs)) {
      // console.log("cur property: ", property);

      // // Handle API call only for ChromeOS and Windows
      // if (property === "protected_content_enabled") {
      //   console.log("protected content property");

      //   // check platform
      //   // TODO - make sure this is being set correctly
      //   let os;
      //   await chrome.runtime.getPlatformInfo((info) => {
      //     console.log("platform info: ", info);
      //     os = info.os;
      //   });

      //   // only call if on ChromeOS or Windows
      //   // TODO - confirm string for windows and chromeos platforms
      //   console.log("os: ", os);
      //   if (os === "windows" || os === "cros") {
      //     // hard to make this modular since can't pass `property` into the callback
      //     console.log("calling chrome/windows only api");
      //     await this.propertyAPIs[property].get({}, (details) => {
      //       if (details.value === null) {
      //         result[property] = null; // TODO: how to handle this? should we handle it?
      //       } else {
      //         // convert bool response to binary flag
      //         result[property] = details.value ? 1 : 0;
      //       }
      //     });
      //   }
      // } else {
      try {
        await propertyAPI.get({}, (details) => {
          // if (details.value === null) {
          //   result[property] = null; // TODO: how to handle this? should we handle it?
          // } else {
          // result[property] = details.value ? 1 : 0;
          // only non-bool property
          if (property === "web_rtc_ip_handling_policy") {
            result[property] = details.value;
          } else {
            // convert bool response to binary flag
            if (details.value === true) {
              result[property] = 1;
            } else {
              result[property] = 0;
              // }
            }
            // }
          }
        });
      } catch (error) {
        console.log("error: ", error);
        console.log("property: ", property);
      }
    }
    console.log("result row: ", result);
    return [result];
  }
}
// }
