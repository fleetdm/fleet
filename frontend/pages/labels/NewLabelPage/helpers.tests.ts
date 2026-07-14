import { CUSTOM_HOST_VITAL_CRITERION } from "interfaces/label";

import {
  buildCriterionOptionValue,
  parseCriterionOptionValue,
  getVitalValuePlaceholder,
  getCriterionHelpText,
} from "./helpers";

describe("NewLabelPage helpers", () => {
  describe("buildCriterionOptionValue / parseCriterionOptionValue", () => {
    it("encodes a custom host vital id into the option value", () => {
      expect(buildCriterionOptionValue(5)).toBe("custom_host_vital:5");
    });

    it("decodes a custom host vital option value back to vital + id", () => {
      expect(parseCriterionOptionValue("custom_host_vital:5")).toEqual({
        vital: CUSTOM_HOST_VITAL_CRITERION,
        customHostVitalId: 5,
      });
    });

    it("decodes an IdP option value with no custom id", () => {
      expect(parseCriterionOptionValue("end_user_idp_group")).toEqual({
        vital: "end_user_idp_group",
      });
      expect(parseCriterionOptionValue("end_user_idp_department")).toEqual({
        vital: "end_user_idp_department",
      });
    });

    it("round-trips any custom host vital id", () => {
      [1, 42, 1000, 999999].forEach((id) => {
        const parsed = parseCriterionOptionValue(buildCriterionOptionValue(id));
        expect(parsed.vital).toBe(CUSTOM_HOST_VITAL_CRITERION);
        expect(parsed.customHostVitalId).toBe(id);
      });
    });

    it("does not treat an IdP value as a custom vital", () => {
      const parsed = parseCriterionOptionValue("end_user_idp_group");
      expect(parsed.vital).toBe("end_user_idp_group");
      expect(parsed.customHostVitalId).toBeUndefined();
    });
  });

  describe("getVitalValuePlaceholder", () => {
    it("returns IdP-specific placeholders", () => {
      expect(getVitalValuePlaceholder("end_user_idp_group")).toBe("IT admins");
      expect(getVitalValuePlaceholder("end_user_idp_department")).toBe(
        "Engineering"
      );
    });

    it("returns a generic placeholder for custom host vitals", () => {
      expect(getVitalValuePlaceholder(CUSTOM_HOST_VITAL_CRITERION)).toBe(
        "Value"
      );
    });
  });

  describe("getCriterionHelpText", () => {
    it("is specific to the selected criterion", () => {
      expect(getCriterionHelpText("end_user_idp_group")).toBe(
        "Label criteria is based on the end user's IdP group."
      );
      expect(getCriterionHelpText("end_user_idp_department")).toBe(
        "Label criteria is based on the end user's IdP department."
      );
      expect(getCriterionHelpText(CUSTOM_HOST_VITAL_CRITERION)).toBe(
        "Label criteria is based on the selected custom host vital."
      );
    });
  });
});
