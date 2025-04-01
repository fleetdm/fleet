import React from "react";
import { Meta, StoryFn } from "@storybook/react";
import { noop } from "lodash";

import { IPaginationProps } from "./Pagination";
import Pagination from ".";

import "../../index.scss";

const meta: Meta<IPaginationProps> = {
  component: Pagination,
  title: "Components/Pagination",
  args: {
    disableNext: false,
    disablePrev: true,
    onNextPage: noop,
    onPrevPage: noop,
    className: "pagination-story",
    hidePagination: false,
  },
};

export default meta;

const Template: StoryFn<IPaginationProps> = (props) => (
  <Pagination {...props} />
);

export const Default = Template.bind({});
