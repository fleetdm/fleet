import VirtualDatabase from "../db";
import set from "lodash/set";

describe("privacy_preferences", () => {
    test("success", async () => {
        const privacyApis = {
            'privacy.network.networkPredictionEnabled': true,
            'privacy.network.webRTCIPHandlingPolicy': "default",

            'privacy.services.autofillAddressEnabled': true,
            'privacy.services.autofillCreditCardEnabled': false,
            'privacy.services.autofillEnabled': true,
            'privacy.services.passwordSavingEnabled': false,
            'privacy.services.safeBrowsingEnabled': true,
            'privacy.services.safeBrowsingExtendedReportingEnabled': false,
            'privacy.services.searchSuggestEnabled': true,
            'privacy.services.spellingServiceEnabled': false,
            'privacy.services.translationServiceEnabled': true,
            'privacy.websites.adMeasurementEnabled': false,
            'privacy.websites.doNotTrackEnabled': true,
            'privacy.websites.fledgeEnabled': false,
            'privacy.websites.hyperlinkAuditingEnabled': true,
            'privacy.websites.privacySandboxEnabled': false,
            'privacy.websites.protectedContentEnabled': true,
            'privacy.websites.referrersEnabled': false,
            'privacy.websites.thirdPartyCookiesAllowed': true,
            'privacy.websites.topicsEnabled': false,
        };
        Object.entries(privacyApis).forEach(([api, value]) => {
            // @ts-ignore
            set(chrome, api, {
                get: jest.fn((_, cb) => {
                    return cb({value})
                }),
                set: jest.fn(),
                clear: jest.fn(),
            });
        });
        
        const db = await VirtualDatabase.init();
        globalThis.DB = db;

        const res = await db.query("select * from privacy_preferences");
        expect(res).toEqual({
            data: [{
                network_prediction_enabled: "1",
                web_rtc_ip_handling_policy: "default",
                autofill_address_enabled: "1",
                autofill_credit_card_enabled: "0",
                autofill_enabled: "1",
                save_passwords_enabled: "0",
                safe_browsing_enabled: "1",
                safe_browsing_extended_reporting_enabled: "0",
                search_suggest_enabled: "1",
                spelling_service_enabled: "0",
                translation_service_enabled: "1",
                ad_measurement_enabled: "0",
                do_not_track_enabled: "1",
                fledge_enabled: "0",
                hyperlink_auditing_enabled: "1",
                privacy_sandbox_enabled: "0",
                protected_content_enabled: "1",
                referrers_enabled: "0",
                third_party_cookies_allowed: "1",
                topics_enabled: "0",
            }
        ],
        warnings: "",
        });
    });    
});

