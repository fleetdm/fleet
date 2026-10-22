import {
  createMockCertificateAuthorityPartial,
  createMockEJBCACertAuthority,
} from "__mocks__/certificatesMock";

import { EMPTY_EJBCA_FORM_DATA, IEJBCAFormData } from "./EJBCAForm";
import {
  generateEJBCAAddData,
  generateEJBCAEditData,
  generateEJBCAFormData,
  getEJBCAErrorMessage,
  readFileAsBase64,
  splitEJBCAServerErrors,
  validateEJBCAForm,
} from "./helpers";

const validFormData: IEJBCAFormData = {
  ...EMPTY_EJBCA_FORM_DATA,
  name: "WIFI_CERTIFICATE",
  url: "https://ejbca.example.com:8443",
  clientP12: "AQID",
  password: "p12-password",
  caName: "ManagementCA",
  certificateProfileName: "ENDUSER",
  endEntityProfileName: "EMPTY",
  commonName: "$FLEET_VAR_HOST_HARDWARE_SERIAL",
};

const context = { certAuthorities: [] };

describe("EJBCAForm helpers", () => {
  describe("validateEJBCAForm", () => {
    it("has no errors for a valid form", () => {
      expect(validateEJBCAForm(validFormData, context)).toEqual({});
    });

    it("has an error for each required field that is empty", () => {
      expect(validateEJBCAForm(EMPTY_EJBCA_FORM_DATA, context)).toEqual({
        name: "Enter a name",
        url: "Enter a URL",
        clientP12: "Upload a client certificate (.p12)",
        caName: "Enter an EJBCA CA name",
        certificateProfileName: "Enter a certificate profile name",
        endEntityProfileName: "Enter an end entity profile name",
        commonName: "Enter a common name",
      });
    });

    it("counts a field of only spaces as empty", () => {
      expect(
        validateEJBCAForm({ ...validFormData, commonName: "  " }, context)
      ).toEqual({ commonName: "Enter a common name" });
    });

    it.each([
      ["has invalid characters", "WIFI CERT", "Enter a name with only"],
      ["is NDES, in any case", "ndes", "Enter a name other than NDES"],
      [
        "is the name of another CA, in any case",
        "test_digicert",
        "Enter a name that isn't used by another certificate authority",
      ],
    ])("has a name error when the name %s", (_, name, error) => {
      const errors = validateEJBCAForm(
        { ...validFormData, name },
        {
          certAuthorities: [
            createMockCertificateAuthorityPartial({
              id: 5,
              name: "TEST_DIGICERT",
              type: "digicert",
            }),
          ],
        }
      );
      expect(errors.name).toContain(error);
    });

    it("lets the CA that is edited keep its name", () => {
      const errors = validateEJBCAForm(
        { ...validFormData, clientP12: "" },
        {
          certAuthorities: [
            createMockCertificateAuthorityPartial({
              id: 5,
              name: validFormData.name,
              type: "ejbca",
            }),
          ],
          editingId: 5,
        }
      );
      expect(errors).toEqual({});
    });

    it("has a URL error for an invalid URL", () => {
      expect(
        validateEJBCAForm({ ...validFormData, url: "not a url" }, context).url
      ).toBe("Enter a valid URL");
    });

    it("needs the password with a client certificate, and the certificate only when adding", () => {
      expect(
        validateEJBCAForm({ ...validFormData, password: "" }, context)
      ).toEqual({ password: "Enter the password of the client certificate" });
      // when editing, the stored certificate stays unless a new one is uploaded
      expect(
        validateEJBCAForm(
          { ...validFormData, clientP12: "", password: "" },
          { certAuthorities: [], editingId: 5 }
        )
      ).toEqual({});
    });

    it("needs a PEM in the trust CA bundle, if there is one", () => {
      expect(
        validateEJBCAForm({ ...validFormData, trustCABundle: "abc" }, context)
          .trustCABundle
      ).toBe("Enter a PEM-encoded certificate");
    });
  });

  describe("generateEJBCAAddData", () => {
    it("maps all the fields", () => {
      expect(
        generateEJBCAAddData({
          ...validFormData,
          trustCABundle: "-----BEGIN CERTIFICATE-----",
          usernameTemplate: "$FLEET_VAR_HOST_UUID",
          userPrincipalName: "upn@example.com",
        })
      ).toEqual({
        name: "WIFI_CERTIFICATE",
        url: "https://ejbca.example.com:8443",
        client_p12: "AQID",
        password: "p12-password",
        trust_ca_bundle: "-----BEGIN CERTIFICATE-----",
        certificate_authority_name: "ManagementCA",
        certificate_profile_name: "ENDUSER",
        end_entity_profile_name: "EMPTY",
        certificate_common_name: "$FLEET_VAR_HOST_HARDWARE_SERIAL",
        username_template: "$FLEET_VAR_HOST_UUID",
        certificate_user_principal_names: ["upn@example.com"],
      });
    });

    it("leaves out the optional fields that are empty, and has no UPNs without one", () => {
      const data = generateEJBCAAddData(validFormData);
      expect(data).not.toHaveProperty("trust_ca_bundle");
      expect(data).not.toHaveProperty("username_template");
      expect(data.certificate_user_principal_names).toEqual([]);
    });
  });

  describe("generateEJBCAEditData", () => {
    const stored = createMockEJBCACertAuthority();
    const storedFormData = generateEJBCAFormData(stored);

    it("has no changes for the stored values", () => {
      expect(generateEJBCAEditData(stored, storedFormData)).toEqual({});
    });

    it("has only the fields that changed", () => {
      expect(
        generateEJBCAEditData(stored, {
          ...storedFormData,
          caName: "OtherCA",
          commonName: "new-cn",
        })
      ).toEqual({
        certificate_authority_name: "OtherCA",
        certificate_common_name: "new-cn",
      });
    });

    it("clears the optional fields that were emptied", () => {
      expect(
        generateEJBCAEditData(stored, {
          ...storedFormData,
          trustCABundle: "",
          usernameTemplate: "",
          userPrincipalName: "",
        })
      ).toEqual({
        trust_ca_bundle: "",
        username_template: "",
        certificate_user_principal_names: [],
      });
    });

    it("sends the client certificate with its password, only when a new one was uploaded", () => {
      expect(
        generateEJBCAEditData(stored, {
          ...storedFormData,
          clientP12: "AQID",
          password: "new-password",
        })
      ).toEqual({ client_p12: "AQID", password: "new-password" });
    });
  });

  describe("splitEJBCAServerErrors", () => {
    it("puts the errors of the fields on the form, without how the server starts and ends them", () => {
      const { fieldErrors, otherReasons } = splitEJBCAServerErrors({
        errors: [
          {
            name: "password",
            reason:
              "Couldn't add certificate authority. The PKCS#12 password is incorrect or the file is damaged.",
          },
          { name: "base", reason: "Couldn't add certificate authority. Oops." },
        ],
      });

      expect(fieldErrors).toEqual({
        password: "The PKCS#12 password is incorrect or the file is damaged",
      });
      expect(otherReasons).toEqual([
        "Couldn't add certificate authority. Oops.",
      ]);
    });
  });

  describe("getEJBCAErrorMessage", () => {
    it("says what failed, with the reasons or what to do", () => {
      expect(
        getEJBCAErrorMessage("add", [
          "Couldn't add certificate authority. EJBCA rejected the TLS client certificate.",
        ])
      ).toBe(
        "Couldn't add certificate authority. EJBCA rejected the TLS client certificate."
      );
      expect(getEJBCAErrorMessage("edit", [])).toBe(
        "Couldn't edit certificate authority. Please try again."
      );
    });
  });

  describe("readFileAsBase64", () => {
    it("reads the bytes of a file as base64", async () => {
      const file = new File([new Uint8Array([1, 2, 3])], "client.p12");
      expect(await readFileAsBase64(file)).toBe("AQID");
    });
  });
});
