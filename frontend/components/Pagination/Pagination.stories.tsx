import React from "react";
import { Meta, StoryFn } from "@storybook/react";
import { noop } from "lodash";

// @ts-ignore
import Pagination from ".";

import "../../index.scss";

interface IPaginationProps {
  currentPage: number;
  resultsPerPage: number;
  resultsOnCurrentPage: number;
  onPaginationChange: () => void;
}

const meta: Meta<IPaginationProps> = {
  component: Pagination,
  title: "Components/Pagination",
  args: {
    currentPage: 1,
    resultsPerPage: 10,
    resultsOnCurrentPage: 10,
    onPaginationChange: noop,
  },
};

export default meta;

const Template: StoryFn<IPaginationProps> = (props) => (
  <Pagination {...props} />
);

export const Default = Template.bind({});
