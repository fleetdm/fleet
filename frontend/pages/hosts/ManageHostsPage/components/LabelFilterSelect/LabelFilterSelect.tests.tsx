import React from "react";
import { screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";
import { createMockLabel } from "__mocks__/labelsMock";
import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";

import LabelFilterSelect from "./LabelFilterSelect";

jest.mock("hooks/useCheckTruncatedElement", () => ({
  useCheckTruncatedElement: jest.fn(),
}));

const mockedUseCheckTruncatedElement = useCheckTruncatedElement as jest.Mock;

const LONG_LABEL_NAME =
  "A very long custom label name that will not fit inside the dropdown";

const defaultProps = {
  labels: [
    createMockLabel({
      id: 42,
      label_type: "regular",
      name: LONG_LABEL_NAME,
      display_text: LONG_LABEL_NAME,
    }),
  ],
  selectedLabel: null,
  canAddNewLabels: true,
  onChange: jest.fn(),
  onAddLabel: jest.fn(),
};

const openMenu = async (user: ReturnType<typeof renderWithSetup>["user"]) => {
  await user.click(screen.getByText("Filter by platform or label"));
};

describe("LabelFilterSelect", () => {
  beforeEach(() => {
    mockedUseCheckTruncatedElement.mockReturnValue(false);
  });

  it("shows a tooltip with the full name when a label option is truncated", async () => {
    mockedUseCheckTruncatedElement.mockReturnValue(true);
    const { user } = renderWithSetup(<LabelFilterSelect {...defaultProps} />);

    await openMenu(user);
    await user.hover(screen.getByText(LONG_LABEL_NAME));

    expect(await screen.findByRole("tooltip")).toHaveTextContent(
      LONG_LABEL_NAME
    );
  });

  it("does not show a tooltip when a label option is not truncated", async () => {
    const { user } = renderWithSetup(<LabelFilterSelect {...defaultProps} />);

    await openMenu(user);
    await user.hover(screen.getByText(LONG_LABEL_NAME));

    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });
});
