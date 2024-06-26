import Table from "./Table";
import ChromeSettingGetResultDetails = chrome.types.ChromeSettingGetResultDetails;

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
    protected_content_enabled: chrome.privacy.websites.protectedContentEnabled,
    referrers_enabled: chrome.privacy.websites.referrersEnabled,
    third_party_cookies_allowed:
      chrome.privacy.websites.thirdPartyCookiesAllowed,
    // @ts-ignore
    topics_enabled: chrome.privacy.websites.topicsEnabled,
  };

  columns = Object.keys(this.propertyAPIs);

  async generate() {
    const results = []; // Promise<{string: number | string}>[]
    const errors = [];
    let warningsArray = [];
    for (const [property, propertyAPI] of Object.entries(this.propertyAPIs)) {
      results.push(
        new Promise((resolve) => {
          try {
            if (propertyAPI === undefined) {
              resolve({ [property]: "" });
            } else {
              propertyAPI.get({}, (details: ChromeSettingGetResultDetails) => {
                if (property === "web_rtc_ip_handling_policy") {
                  resolve({ [property]: details.value });
                } else {
                  // bool responses converted to binary flag in upper layer
                  resolve({ [property]: details.value });
                }
              });
            }
          } catch (error) {
            errors.push({ [property]: error });
            warningsArray.push({
              column: property,
              error_message: error.stack.toString(),
            });
            resolve({ [property]: "data unavailable" });
          }
        })
      );
    }

    // wait for each API call to resolve
    const columns = await Promise.all(results);
    errors.length > 0 &&
      console.log("Caught errors in chrome API calls: ", errors);
    return {
      data: [
        columns.reduce((resultRow, column) => {
          return { ...resultRow, ...column };
        }, {}),
      ],
      warnings: warningsArray,
    };
  }
}
