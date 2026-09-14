import { createMockConfig } from "__mocks__/configMock";

import { ISsoFormData, newSsoFormData, validateSsoForm } from "./helpers";

const VALID_FORM_DATA: ISsoFormData = {
  enableSso: true,
  idpName: "Okta",
  entityId: "https://fleet.example.com",
  idpImageUrl: "",
  metadata: "",
  metadataUrl: "https://idp.example.com/metadata",
  enableSsoIdpLogin: false,
  enableJitProvisioning: false,
};

describe("Sso helpers", () => {
  describe("validateSsoForm", () => {
    it("returns no errors for a valid configuration", () => {
      expect(validateSsoForm(VALID_FORM_DATA)).toEqual({});
    });

    it("skips every check while SSO is turned off", () => {
      expect(
        validateSsoForm({
          ...VALID_FORM_DATA,
          enableSso: false,
          idpName: "",
          entityId: "",
          metadata: "",
          metadataUrl: "",
          idpImageUrl: "not a url",
        })
      ).toEqual({});
    });

    it("requires the identity provider name and entity ID", () => {
      expect(
        validateSsoForm({ ...VALID_FORM_DATA, idpName: "", entityId: "" })
      ).toEqual({
        idpName: "Enter an identity provider name",
        entityId: "Enter an entity ID",
      });
    });

    it("treats whitespace-only required fields as empty", () => {
      expect(
        validateSsoForm({ ...VALID_FORM_DATA, idpName: "   ", entityId: " " })
      ).toEqual({
        idpName: "Enter an identity provider name",
        entityId: "Enter an entity ID",
      });
    });

    it("requires either metadata or a metadata URL", () => {
      expect(
        validateSsoForm({ ...VALID_FORM_DATA, metadata: "", metadataUrl: "" })
      ).toEqual({
        metadata: "Enter metadata or a metadata URL",
        metadataUrl: "Enter metadata or a metadata URL",
      });
    });

    it("accepts metadata on its own", () => {
      expect(
        validateSsoForm({
          ...VALID_FORM_DATA,
          metadata: "<xml />",
          metadataUrl: "",
        })
      ).toEqual({});
    });

    it("rejects an invalid metadata URL even when metadata is also present", () => {
      expect(
        validateSsoForm({
          ...VALID_FORM_DATA,
          metadata: "<xml />",
          metadataUrl: "not a url",
        })
      ).toEqual({ metadataUrl: "Enter a valid metadata URL" });
    });

    it("rejects a metadata URL without a supported protocol", () => {
      expect(
        validateSsoForm({
          ...VALID_FORM_DATA,
          metadataUrl: "idp.example.com/metadata",
        })
      ).toEqual({
        metadataUrl: "Enter a metadata URL starting with https:// or http://",
      });
    });

    it("validates the IdP image URL only when it has a value", () => {
      expect(validateSsoForm({ ...VALID_FORM_DATA, idpImageUrl: "" })).toEqual(
        {}
      );

      expect(
        validateSsoForm({ ...VALID_FORM_DATA, idpImageUrl: "not a url" })
      ).toEqual({ idpImageUrl: "Enter a valid IdP image URL" });
    });
  });

  describe("newSsoFormData", () => {
    it("trims the saved values", () => {
      const formData = newSsoFormData({
        ...createMockConfig(),
        sso_settings: {
          enable_sso: true,
          idp_name: "  Okta  ",
          entity_id: "  https://fleet.example.com  ",
          idp_image_url: "",
          metadata: "",
          metadata_url: "  https://idp.example.com/metadata  ",
          issuer_uri: "",
          enable_sso_idp_login: false,
          enable_jit_provisioning: false,
          enable_jit_role_sync: false,
        },
      });

      expect(formData).toMatchObject({
        enableSso: true,
        idpName: "Okta",
        entityId: "https://fleet.example.com",
        metadataUrl: "https://idp.example.com/metadata",
      });
    });

    it("falls back to empty strings when nothing is configured", () => {
      expect(newSsoFormData(createMockConfig())).toMatchObject({
        idpName: "",
        entityId: "",
        idpImageUrl: "",
        metadata: "",
        metadataUrl: "",
      });
    });
  });
});
