import React from "react";
import { Meta, Story } from "@storybook/react";
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

export default {
  component: Pagination,
  title: "Components/Pagination",
  args: {
    currentPage: 1,
    resultsPerPage: 10,
    resultsOnCurrentPage: 10,
    onPaginationChange: noop,
  },
} as Meta;

const Template: Story<IPaginationProps> = (props) => <Pagination {...props} />;

export const Default = Template.bind({});
