import React, { MutableRefObject } from "react";
import { noop } from "lodash";
import { screen } from "@testing-library/react";

import { createMockConfig } from "__mocks__/configMock";
import { createCustomRenderer } from "test/test-utils";

import EndUserAuthSection, {
  IEndUserAuthSectionProps,
} from "./EndUserAuthSection";
import { IFormDataIdp } from "./helpers";

const EMPTY_FORM_DATA: IFormDataIdp = {
  idp_name: "",
  entity_id: "",
  metadata_url: "",
  metadata: "",
};

const FILLED_FORM_DATA: IFormDataIdp = {
  idp_name: "Okta",
  entity_id: "https://fleet.example.com",
  metadata_url: "https://idp.example.com/metadata",
  metadata: "",
};

const createTestRenderer = () => {
  return createCustomRenderer({
    context: {
      app: {
        isPremiumTier: true,
        config: createMockConfig(),
      },
      notification: {
        renderFlash: jest.fn(),
      },
    },
  });
};

const renderEndUserAuthSection = (
  overrides?: Partial<IEndUserAuthSectionProps>
) => {
  const defaultProps: IEndUserAuthSectionProps = {
    setDirty: jest.fn(),
    formData: FILLED_FORM_DATA,
    setFormData: jest.fn(),
    originalFormData: {
      current: FILLED_FORM_DATA,
    } as MutableRefObject<IFormDataIdp>,
    onSubmit: noop,
    ...overrides,
  };

  const render = createTestRenderer();
  return render(<EndUserAuthSection {...defaultProps} />);
};

describe("EndUserAuthSection", () => {
  it("enables Save when all fields are cleared but original data had values", () => {
    renderEndUserAuthSection({
      formData: EMPTY_FORM_DATA,
      originalFormData: {
        current: FILLED_FORM_DATA,
      } as MutableRefObject<IFormDataIdp>,
    });

    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
  });

  it("disables Save when form starts empty and remains empty", () => {
    renderEndUserAuthSection({
      formData: EMPTY_FORM_DATA,
      originalFormData: {
        current: EMPTY_FORM_DATA,
      } as MutableRefObject<IFormDataIdp>,
    });

    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("enables Save when all required fields are filled with valid data", () => {
    renderEndUserAuthSection({
      formData: FILLED_FORM_DATA,
      originalFormData: {
        current: FILLED_FORM_DATA,
      } as MutableRefObject<IFormDataIdp>,
    });

    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
  });

  it("disables Save when required fields are partially filled", () => {
    renderEndUserAuthSection({
      formData: {
        idp_name: "Okta",
        entity_id: "",
        metadata_url: "",
        metadata: "",
      },
      originalFormData: {
        current: FILLED_FORM_DATA,
      } as MutableRefObject<IFormDataIdp>,
    });

    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("clears form errors when user clears all fields", async () => {
    const setFormData = jest.fn();
    const { user } = renderEndUserAuthSection({
      formData: FILLED_FORM_DATA,
      setFormData,
      originalFormData: {
        current: FILLED_FORM_DATA,
      } as MutableRefObject<IFormDataIdp>,
    });

    // Clear the Identity provider name field
    const idpNameInput = screen.getByLabelText("Identity provider name");
    await user.clear(idpNameInput);

    // setFormData should have been called with idp_name cleared
    expect(setFormData).toHaveBeenCalledWith(
      expect.objectContaining({ idp_name: "" })
    );
  });
});
