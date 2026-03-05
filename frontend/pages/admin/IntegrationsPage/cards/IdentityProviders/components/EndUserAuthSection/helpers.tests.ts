import { IEndUserAuthentication } from "interfaces/config";
import {
  isEmptyFormData,
  isMissingAnyRequiredField,
  newFormDataIdp,
  validateFormDataIdp,
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

  describe("isMissingAnyRequiredField", () => {
    it("returns true if missing any required field", () => {
      expect(
        isMissingAnyRequiredField({
          entity_id: "entityId",
          idp_name: "idpImageUrl",
          metadata: "metadata",
          metadata_url: "metadataUrl",
        })
      ).toBe(false); // all fields present

      expect(
        isMissingAnyRequiredField({
          entity_id: "",
          idp_name: "idpImageUrl",
          metadata: "metadata",
          metadata_url: "metadataUrl",
        })
      ).toBe(true); // entity_id is missing

      expect(
        isMissingAnyRequiredField({
          entity_id: "entityId",
          idp_name: "",
          metadata: "metadata",
          metadata_url: "metadataUrl",
        })
      ).toBe(true); // idp_name is missing

      expect(
        isMissingAnyRequiredField({
          entity_id: "entityId",
          idp_name: "idpImageUrl",
          metadata: "",
          metadata_url: "",
        })
      ).toBe(true); // metadata or metadata_url must be present
    });

    expect(
      isMissingAnyRequiredField({
        entity_id: "entityId",
        idp_name: "idpImageUrl",
        metadata: "",
        metadata_url: "metadataUrl",
      })
    ).toBe(false); // metadata is not required if metadata_url is present

    expect(
      isMissingAnyRequiredField({
        entity_id: "entityId",
        idp_name: "idpImageUrl",
        metadata: "metadata",
        metadata_url: "",
      })
    ).toBe(false); // metadata_url is not required if metadata is present
  });

  describe("validateFormDataIdP", () => {
    it("returns expected error messages", () => {
      expect(
        validateFormDataIdp({
          entity_id: "entityId",
          idp_name: "idpImageUrl",
          metadata: "metadata",
          metadata_url: "https://metadataUrl.com",
        })
      ).toEqual(null); // all fields valid

      expect(
        validateFormDataIdp({
          entity_id: "",
          idp_name: "",
          metadata: "",
          metadata_url: "",
        })
      ).toEqual(null); // all fields empty is valid (allows clearing settings)

      expect(
        validateFormDataIdp({
          entity_id: "entityId",
          idp_name: "",
          metadata: "metadata",
          metadata_url: "https://metadataUrl.com",
        })
      ).toEqual({
        idp_name: "Identity provider name must be present.",
      });

      expect(
        validateFormDataIdp({
          entity_id: "",
          idp_name: "idpImageUrl",
          metadata: "metadata",
          metadata_url: "https://metadataUrl.com",
        })
      ).toEqual({
        entity_id: "Entity ID must be present.",
      });

      expect(
        validateFormDataIdp({
          entity_id: "entityId",
          idp_name: "idpImageUrl",
          metadata: "",
          metadata_url: "",
        })
      ).toEqual({
        metadata: "Metadata or Metadata URL must be present.",
        metadata_url: "Metadata or Metadata URL must be present.",
      });

      expect(
        validateFormDataIdp({
          entity_id: "entityId",
          idp_name: "idpImageUrl",
          metadata: "metadata",
          metadata_url: "metadataUrl",
        })
      ).toEqual({
        metadata_url: "Metadata URL is not a valid URL.",
      });

      expect(
        validateFormDataIdp({
          entity_id: "entityId",
          idp_name: "idpImageUrl",
          metadata: "metadata",
          metadata_url: "metadataUrl.com",
        })
      ).toEqual({
        metadata_url:
          "Metadata URL must start with a supported protocol (https:// or http://).",
      });

      expect(
        validateFormDataIdp({
          entity_id: "entityId",
          idp_name: "idpImageUrl",
          metadata: "metadata",
          metadata_url: "",
        })
      ).toEqual(null); // metadata is not required if metadata_url is present

      expect(
        validateFormDataIdp({
          entity_id: "entityId",
          idp_name: "idpImageUrl",
          metadata: "",
          metadata_url: "https://metadataUrl.com",
        })
      ).toEqual(null); // metadata is not required if metadata_url is present
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
