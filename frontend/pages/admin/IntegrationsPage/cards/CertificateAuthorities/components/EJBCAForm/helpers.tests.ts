import {
  createMockCertificateAuthorityPartial,
  createMockEJBCACertAuthority,
} from "__mocks__/certificatesMock";

import { EMPTY_EJBCA_FORM_DATA, IEJBCAFormData } from "./EJBCAForm";
import {
  generateEJBCAAddData,
  generateEJBCAEditData,
  generateEJBCAFormData,
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
    it("has no errors for a valid form, including the URL of an internal host", () => {
      expect(validateEJBCAForm(validFormData, context)).toEqual({});
      expect(
        validateEJBCAForm(
          { ...validFormData, url: "https://ejbca:8443" },
          context
        )
      ).toEqual({});
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

    it("lets the CA that is edited keep its name and stored client certificate", () => {
      const errors = validateEJBCAForm(
        { ...validFormData, clientP12: "", password: "" },
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

    it.each([
      ["is not a URL", "not a url", "Enter a valid URL"],
      [
        "is http",
        "http://ejbca.example.com",
        "Enter an https URL without a username or password",
      ],
      [
        "has credentials",
        "https://user:pass@ejbca.example.com",
        "Enter an https URL without a username or password",
      ],
    ])("has a URL error when the URL %s", (_, url, error) => {
      expect(validateEJBCAForm({ ...validFormData, url }, context).url).toBe(
        error
      );
    });

    it("needs the password with a client certificate", () => {
      expect(
        validateEJBCAForm({ ...validFormData, password: "" }, context)
      ).toEqual({ password: "Enter the password of the client certificate" });
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

    it("doesn't count the whitespace the form trims as a change", () => {
      // a PEM, as GitOps stores it, ends with a newline
      const pem =
        "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n";
      const storedWithPem = createMockEJBCACertAuthority({
        trust_ca_bundle: pem,
      });

      expect(
        generateEJBCAEditData(storedWithPem, {
          ...generateEJBCAFormData(storedWithPem),
          trustCABundle: pem.trim(),
          name: "NEW_NAME",
        })
      ).toEqual({ name: "NEW_NAME" });
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
    it("maps the names the server reports to the fields of the form, and leaves the rest apart", () => {
      const { fieldErrors, otherReasons } = splitEJBCAServerErrors({
        errors: [
          { name: "password", reason: "The password is incorrect." },
          { name: "certificate_authority_name", reason: "Unknown CA." },
          { name: "certificate_user_principal_names", reason: "Bad UPN." },
          { name: "base", reason: "Oops." },
        ],
      });

      expect(fieldErrors).toEqual({
        password: "The password is incorrect.",
        caName: "Unknown CA.",
        userPrincipalName: "Bad UPN.",
      });
      expect(otherReasons).toEqual(["Oops."]);
    });
  });

  describe("readFileAsBase64", () => {
    it("reads the bytes of a file as base64", async () => {
      const file = new File([new Uint8Array([1, 2, 3])], "client.p12");
      expect(await readFileAsBase64(file)).toBe("AQID");
    });
  });
});
