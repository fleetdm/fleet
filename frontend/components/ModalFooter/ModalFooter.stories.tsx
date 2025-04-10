/* eslint-disable no-alert */
import React from "react";
import { Meta, StoryObj } from "@storybook/react";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import ActionsDropdown from "components/ActionsDropdown";
import ModalFooter from "./ModalFooter";

const meta: Meta<typeof ModalFooter> = {
  title: "Components/ModalFooter",
  component: ModalFooter,
};

export default meta;

type Story = StoryObj<typeof ModalFooter>;

export const Default: Story = {
  args: {
    primaryButtons: (
      <>
        <ActionsDropdown
          className="modal-footer__manage-automations-dropdown"
          onChange={(value) => alert(`Selected action: ${value}`)}
          placeholder="More actions"
          isSearchable={false}
          options={[
            { value: "action1", label: "Action 1" },
            { value: "action2", label: "Action 2" },
          ]}
          menuPlacement="top"
        />
        <Button onClick={() => alert("Done clicked")} variant="brand">
          Done
        </Button>
      </>
    ),
    secondaryButtons: (
      <>
        <Button variant="icon" onClick={() => alert("Download clicked")}>
          <Icon name="download" />
        </Button>
        <Button variant="icon" onClick={() => alert("Delete clicked")}>
          <Icon name="trash" color="ui-fleet-black-75" />
        </Button>
      </>
    ),
    isTopScrolling: false,
  },
};

export const WithTopScrolling: Story = {
  args: {
    ...Default.args,
    isTopScrolling: true,
  },
};

export const WithoutSecondaryButtons: Story = {
  args: {
    primaryButtons: Default.args?.primaryButtons,
    secondaryButtons: undefined,
    isTopScrolling: false,
  },
};
