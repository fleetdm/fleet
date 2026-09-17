import { IEndUserAuthentication } from "interfaces/config";
import {
  isEmptyFormData,
  newFormDataIdp,
  validateEndUserAuthForm,
} from "./helpers";

describe("IdPSection helpers", () => {
  describe("isEmptyFormData", () => {
    it("returns true when all fields are empty", () => {
      expect(
        isEmptyFormData({
          entity_id: "",
          idp_name: "",
          metadata: "",
          metadata_url: "",
        })
      ).toBe(true);
    });

    it("returns false when any field is non-empty", () => {
      expect(
        isEmptyFormData({
          entity_id: "entityId",
          idp_name: "",
          metadata: "",
          metadata_url: "",
        })
      ).toBe(false);

      expect(
        isEmptyFormData({
          entity_id: "",
          idp_name: "idpName",
          metadata: "",
          metadata_url: "",
        })
      ).toBe(false);
    });
  });

  describe("validateEndUserAuthForm", () => {
    it("returns no errors when all fields are valid", () => {
      expect(
        validateEndUserAuthForm({
          entity_id: "entityId",
          idp_name: "idpName",
          metadata: "metadata",
          metadata_url: "https://metadataUrl.com",
        })
      ).toEqual({});
    });

    it("returns no errors when every field is empty, so the config can be cleared", () => {
      expect(
        validateEndUserAuthForm({
          entity_id: "",
          idp_name: "",
          metadata: "",
          metadata_url: "",
        })
      ).toEqual({});
    });

    it("treats whitespace-only fields as empty", () => {
      expect(
        validateEndUserAuthForm({
          entity_id: "  ",
          idp_name: "   ",
          metadata: " ",
          metadata_url: "  ",
        })
      ).toEqual({});

      expect(
        validateEndUserAuthForm({
          entity_id: "entityId",
          idp_name: "   ",
          metadata: "metadata",
          metadata_url: "",
        })
      ).toEqual({ idp_name: "Enter an identity provider name" });
    });

    it("requires the identity provider name", () => {
      expect(
        validateEndUserAuthForm({
          entity_id: "entityId",
          idp_name: "",
          metadata: "metadata",
          metadata_url: "https://metadataUrl.com",
        })
      ).toEqual({ idp_name: "Enter an identity provider name" });
    });

    it("requires the entity ID", () => {
      expect(
        validateEndUserAuthForm({
          entity_id: "",
          idp_name: "idpName",
          metadata: "metadata",
          metadata_url: "https://metadataUrl.com",
        })
      ).toEqual({ entity_id: "Enter an entity ID" });
    });

    it("requires either metadata or a metadata URL", () => {
      expect(
        validateEndUserAuthForm({
          entity_id: "entityId",
          idp_name: "idpName",
          metadata: "",
          metadata_url: "",
        })
      ).toEqual({
        metadata: "Enter metadata or a metadata URL",
        metadata_url: "Enter metadata or a metadata URL",
      });
    });

    it("accepts either metadata or a metadata URL on its own", () => {
      expect(
        validateEndUserAuthForm({
          entity_id: "entityId",
          idp_name: "idpName",
          metadata: "metadata",
          metadata_url: "",
        })
      ).toEqual({});

      expect(
        validateEndUserAuthForm({
          entity_id: "entityId",
          idp_name: "idpName",
          metadata: "",
          metadata_url: "https://metadataUrl.com",
        })
      ).toEqual({});
    });

    it("rejects a malformed metadata URL", () => {
      expect(
        validateEndUserAuthForm({
          entity_id: "entityId",
          idp_name: "idpName",
          metadata: "metadata",
          metadata_url: "metadataUrl",
        })
      ).toEqual({ metadata_url: "Enter a valid metadata URL" });
    });

    it("rejects a metadata URL without a supported protocol", () => {
      expect(
        validateEndUserAuthForm({
          entity_id: "entityId",
          idp_name: "idpName",
          metadata: "metadata",
          metadata_url: "metadataUrl.com",
        })
      ).toEqual({
        metadata_url: "Enter a metadata URL starting with https:// or http://",
      });
    });
  });

  describe("newFormDataIdP", () => {
    it("returns expected new form data", () => {
      expect(
        newFormDataIdp({
          entity_id: "entityId",
          idp_name: "idpImageUrl",
          issuer_uri: "issuerUri",
          metadata: "metadata",
          metadata_url: "https://metadataUrl.com",
        })
      ).toEqual({
        entity_id: "entityId",
        idp_name: "idpImageUrl",
        metadata: "metadata",
        metadata_url: "https://metadataUrl.com",
      }); // all fields valid
    });

    expect(
      newFormDataIdp({
        entity_id: "entityId   ",
        idp_name: "    idpImageUrl",
        issuer_uri: "issuerUri",
        metadata: "metadata",
        metadata_url: "   https://metadataUrl.com   ",
      })
    ).toEqual({
      entity_id: "entityId",
      idp_name: "idpImageUrl",
      metadata: "metadata",
      metadata_url: "https://metadataUrl.com",
    }); // whitespace trimmed

    expect(newFormDataIdp(undefined)).toEqual({
      entity_id: "",
      idp_name: "",
      metadata: "",
      metadata_url: "",
    }); // all fields missing

    expect(
      newFormDataIdp({
        entity_id: "entityId",
      } as IEndUserAuthentication)
    ).toEqual({
      entity_id: "entityId",
      idp_name: "",
      metadata: "",
      metadata_url: "",
    }); // idp_name, metadata, metadata_url missing
  });
});
