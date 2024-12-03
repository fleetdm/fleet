/* eslint-disable no-alert */
import React from "react";
import { Meta, Story } from "@storybook/react";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import ActionsDropdown from "components/ActionsDropdown";
import ModalFooter from "./ModalFooter";

export default {
  title: "Components/ModalFooter",
  component: ModalFooter,
} as Meta;

const Template: Story = (args) => (
  <ModalFooter primaryButtons={<></>} {...args} />
);

export const Default = Template.bind({});
Default.args = {
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
};

export const WithTopScrolling = Template.bind({});
WithTopScrolling.args = {
  ...Default.args,
  isTopScrolling: true,
};

export const WithoutSecondaryButtons = Template.bind({});
WithoutSecondaryButtons.args = {
  primaryButtons: Default.args.primaryButtons,
  secondaryButtons: undefined,
  isTopScrolling: false,
};
