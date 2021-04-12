import { configStub } from "test/stubs";
import helpers from "redux/nodes/app/helpers";

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

      expect(frontendFormattedConfig(configStub)).toEqual({
        ...orgInfo,
        ...serverSettings,
        ...smtpSettings,
        ...hostExpirySettings,
      });
    });
  });
});
