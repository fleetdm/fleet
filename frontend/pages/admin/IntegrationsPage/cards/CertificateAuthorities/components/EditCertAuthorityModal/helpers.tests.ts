import {
  createMockGoogleCASCertAuthority,
  createMockNDESCertAuthority,
} from "__mocks__/certificatesMock";
import { UNCHANGED_PASSWORD_API_RESPONSE } from "utilities/constants";

import { IGoogleCASFormData } from "../GoogleCASForm/GoogleCASForm";
import { INDESFormData } from "../NDESForm/NDESForm";

import {
  generateDefaultFormData,
  generateEditCertAuthorityData,
  updateFormData,
} from "./helpers";

const ndesCertAuthority = createMockNDESCertAuthority();

const ndesFormData: INDESFormData = {
  scepURL: ndesCertAuthority.url,
  adminURL: ndesCertAuthority.admin_url,
  username: ndesCertAuthority.username,
  password: UNCHANGED_PASSWORD_API_RESPONSE,
};

describe("EditCertAuthorityModal helpers", () => {
  describe("updateFormData", () => {
    // The NDES credentials are validated together against the NDES server, so any change
    // to a field they're validated with has to be sent with the password.
    it.each(["scepURL", "adminURL", "username"])(
      "clears the unchanged NDES password when %s changes",
      (fieldName) => {
        const newFormData = updateFormData(ndesCertAuthority, ndesFormData, {
          name: fieldName,
          value: "new value",
        }) as INDESFormData;

        expect(newFormData.password).toBe("");
      }
    );

    it("keeps an already entered NDES password when the SCEP URL changes", () => {
      const newFormData = updateFormData(
        ndesCertAuthority,
        { ...ndesFormData, password: "entered-password" },
        { name: "scepURL", value: "https://new.example.com" }
      ) as INDESFormData;

      expect(newFormData.password).toBe("entered-password");
    });
  });

  describe("generateEditCertAuthorityData", () => {
    it("includes the password with an NDES SCEP URL change", () => {
      const formData: INDESFormData = {
        ...ndesFormData,
        scepURL: "https://new.example.com/certsrv/mscep/mscep.dll",
        password: "entered-password",
      };

      expect(
        generateEditCertAuthorityData(ndesCertAuthority, formData)
      ).toEqual({
        ndes_scep_proxy: {
          url: "https://new.example.com/certsrv/mscep/mscep.dll",
          password: "entered-password",
        },
      });
    });

    it("sends only the password when only the NDES password changes", () => {
      const formData: INDESFormData = {
        ...ndesFormData,
        password: "entered-password",
      };

      expect(
        generateEditCertAuthorityData(ndesCertAuthority, formData)
      ).toEqual({
        ndes_scep_proxy: { password: "entered-password" },
      });
    });
  });
});

describe("EditCertAuthorityModal helpers for Google CAS", () => {
  const googleCASCertAuthority = createMockGoogleCASCertAuthority();
  const googleCASFormData = generateDefaultFormData(
    googleCASCertAuthority
  ) as IGoogleCASFormData;

  it("fills the form from the saved CA, keeping the key masked", () => {
    expect(googleCASFormData).toEqual({
      name: "GOOGLE_WIFI",
      projectId: "my-project",
      caPool: "my-ca-pool",
      location: "us-central1",
      apiKeyJson: UNCHANGED_PASSWORD_API_RESPONSE,
    });
  });

  it("keeps the masked key when the pool changes, since the server verifies with the saved key", () => {
    const newFormData = updateFormData(
      googleCASCertAuthority,
      googleCASFormData,
      { name: "caPool", value: "other-pool" }
    ) as IGoogleCASFormData;

    expect(newFormData).toEqual({ ...googleCASFormData, caPool: "other-pool" });
  });

  it("sends only the changed fields, leaving out an unchanged key", () => {
    expect(
      generateEditCertAuthorityData(googleCASCertAuthority, {
        ...googleCASFormData,
        caPool: "other-pool",
      })
    ).toEqual({ google_cas: { ca_pool: "other-pool" } });
  });

  it("sends a replaced key", () => {
    expect(
      generateEditCertAuthorityData(googleCASCertAuthority, {
        ...googleCASFormData,
        apiKeyJson: '{"type":"service_account"}',
      })
    ).toEqual({ google_cas: { api_key_json: '{"type":"service_account"}' } });
  });
});
