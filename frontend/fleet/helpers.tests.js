import { omit } from "lodash";

import { configStub, scheduledQueryStub } from "test/stubs";
import helpers from "fleet/helpers";

const label1 = { id: 1, target_type: "labels" };
const label2 = { id: 2, target_type: "labels" };
const host1 = { id: 6, target_type: "hosts" };
const host2 = { id: 5, target_type: "hosts" };

describe("Fleet API - helpers", () => {
  describe("#labelSlug", () => {
    it("creates a slug for the label", () => {
      expect(helpers.labelSlug({ name: "All Hosts" })).toEqual("all-hosts");
      expect(helpers.labelSlug({ id: 7 })).toEqual("labels/7");
    });
  });

  describe("#formatConfigDataForServer", () => {
    const { formatConfigDataForServer } = helpers;
    const config = {
      org_name: "Fleet",
      org_logo_url: "0.0.0.0:8080/logo.png",
      server_url: "",
      configured: false,
      domain: "",
      smtp_enabled: true,
      sender_address: "",
      server: "",
      port: 587,
      authentication_type: "authtype_username_password",
      user_name: "",
      password: "",
      enable_ssl_tls: true,
      authentication_method: "authmethod_plain",
      verify_ssl_certs: true,
      enable_start_tls: true,
      host_expiry_enabled: false,
      host_expiry_window: 0,
      live_query_disabled: false,
      enable_host_status_webhook: false,
      destination_url: "http://server.com/example",
      host_percentage: 5,
      days_count: 7,
    };

    it("splits config into categories for the server", () => {
      expect(formatConfigDataForServer(config)).toEqual(
        omit(configStub, ["smtp_settings.configured"])
      );
      expect(formatConfigDataForServer({ org_name: "The Gnar Co" })).toEqual({
        org_info: { org_name: "The Gnar Co" },
      });
      expect(
        formatConfigDataForServer({
          org_name: "The Gnar Co",
          server_url: "https://example.com",
        })
      ).toEqual({
        org_info: { org_name: "The Gnar Co" },
        server_settings: { server_url: "https://example.com" },
      });
      expect(formatConfigDataForServer({ domain: "https://Fleet.co" })).toEqual(
        {
          smtp_settings: { domain: "https://Fleet.co" },
        }
      );
      expect(formatConfigDataForServer({ host_expiry_window: "12" })).toEqual({
        host_expiry_settings: {
          host_expiry_window: 12,
        },
      });
    });
  });

  describe("#formatScheduledQueryForServer", () => {
    const { formatScheduledQueryForServer } = helpers;
    const scheduledQuery = {
      ...scheduledQueryStub,
      logging_type: "snapshot",
      pack_id: "3",
      platform: "all",
      query_id: "1",
      shard: "12",
    };

    it("sets the correct attributes for the server", () => {
      expect(formatScheduledQueryForServer(scheduledQuery)).toEqual({
        ...scheduledQueryStub,
        pack_id: 3,
        platform: "",
        query_id: 1,
        shard: 12,
        snapshot: true,
        removed: false,
      });
    });
  });

  describe("#formatScheduledQueryForClient", () => {
    const { formatScheduledQueryForClient } = helpers;
    const scheduledQuery = {
      ...scheduledQueryStub,
      platform: "",
      snapshot: true,
    };

    it("sets the correct attributes for the server", () => {
      expect(formatScheduledQueryForClient(scheduledQuery)).toEqual({
        ...scheduledQueryStub,
        logging_type: "snapshot",
        platform: "all",
      });
    });

    it("sets the logging_type attribute", () => {
      expect(
        formatScheduledQueryForClient({
          ...scheduledQueryStub,
          removed: true,
          snapshot: false,
        })
      ).toEqual({
        ...scheduledQueryStub,
        logging_type: "differential",
        removed: true,
        snapshot: false,
      });

      expect(
        formatScheduledQueryForClient({ ...scheduledQueryStub, snapshot: true })
      ).toEqual({
        ...scheduledQueryStub,
        logging_type: "snapshot",
        snapshot: true,
      });

      expect(
        formatScheduledQueryForClient({
          ...scheduledQueryStub,
          snapshot: false,
          removed: false,
        })
      ).toEqual({
        ...scheduledQueryStub,
        logging_type: "differential_ignore_removals",
        removed: false,
        snapshot: false,
      });
    });
  });

  describe("#formatSelectedTargetsForApi", () => {
    const { formatSelectedTargetsForApi } = helpers;

    it("splits targets into labels, hosts, and teams", () => {
      const targets = [host1, host2, label1, label2];

      expect(formatSelectedTargetsForApi(targets)).toEqual({
        hosts: [6, 5],
        labels: [1, 2],
        teams: [],
      });
    });

    it("appends `_id` when appendID is specified", () => {
      const targets = [host1, host2, label1, label2];

      expect(formatSelectedTargetsForApi(targets, true)).toEqual({
        host_ids: [6, 5],
        label_ids: [1, 2],
        team_ids: [],
      });
    });
  });

  describe("#setupData", () => {
    const formData = {
      email: "hi@gnar.dog",
      name: "Gnar Dog",
      server_url: "https://gnar.Fleet.co",
      org_logo_url: "https://thegnar.co/assets/logo.png",
      org_name: "The Gnar Co.",
      password: "p@ssw0rd",
      password_confirmation: "p@ssw0rd",
    };

    it("formats the form data to send to the server", () => {
      expect(helpers.setupData(formData)).toEqual({
        server_url: "https://gnar.Fleet.co",
        org_info: {
          org_logo_url: "https://thegnar.co/assets/logo.png",
          org_name: "The Gnar Co.",
        },
        admin: {
          admin: true,
          email: "hi@gnar.dog",
          name: "Gnar Dog",
          password: "p@ssw0rd",
          password_confirmation: "p@ssw0rd",
        },
      });
    });
  });
});

describe("redux app node - helpers", () => {
  describe("#frontendFormattedConfig", () => {
    const { frontendFormattedConfig } = helpers;

    it("returns a flattened config object", () => {
      const {
        org_info: orgInfo,
        server_settings: serverSettings,
        smtp_settings: smtpSettings,
        host_expiry_settings: hostExpirySettings,
      } = configStub;
      const {
        host_status_webhook: webhookSettings,
      } = configStub.webhook_settings;

      expect(frontendFormattedConfig(configStub)).toEqual({
        ...orgInfo,
        ...serverSettings,
        ...smtpSettings,
        ...hostExpirySettings,
        ...webhookSettings,
      });
    });
  });
});

describe("conversion - helpers", () => {
  describe("#secondsConversions", () => {
    const { secondsToHms, secondsToDhms } = helpers;

    it("creates correct conversion to hour-minute-second", () => {
      expect(secondsToHms(10861)).toEqual("3 hrs 1 min 1 sec");
      expect(secondsToHms(3723)).toEqual("1 hr 2 mins 3 secs");
    });

    it("creates correct conversion to day-hour-minute-second", () => {
      expect(secondsToDhms(270061)).toEqual("3 days 3 hours 1 minute 1 second");
      expect(secondsToDhms(90123)).toEqual("1 day 1 hour 2 minutes 3 seconds");
    });
  });
});
