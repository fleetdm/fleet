// components/List.stories.tsx
import type { Meta, StoryObj } from "@storybook/react";
import React from "react";

import List, { IListProps } from "./List";

interface IStoryItem {
  id: number;
  name: string;
  detail?: string;
}

const meta: Meta<IListProps<IStoryItem>> = {
  title: "Components/List",
  component: List,
  args: {
    data: [
      { id: 1, name: "First item", detail: "Some extra details" },
      { id: 2, name: "Second item", detail: "Other details" },
      { id: 3, name: "Third item" },
    ],
    renderItemRow: (item: IStoryItem) => (
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          width: "100%",
        }}
      >
        <span>{item.name}</span>
        {item.detail && <span>{item.detail}</span>}
      </div>
    ),
  } as Partial<IListProps<IStoryItem>>,
};

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithHeading: Story = {
  args: {
    heading: <div>Items heading</div>,
  },
};

export const WithHelpText: Story = {
  args: {
    helpText: "This is some contextual help text below the list.",
  },
};

export const ClickableRows: Story = {
  args: {
    onClickRow: (item: IStoryItem) => {
      // eslint-disable-next-line no-console
      console.log("Row clicked:", item);
    },
  },
};

export const Loading: Story = {
  args: {
    isLoading: true,
  },
};

export const CustomIdKey: Story = {
  render: () => {
    interface CustomItem {
      customId: string;
      name: string;
    }

    const data: CustomItem[] = [
      { customId: "alpha", name: "Alpha" },
      { customId: "beta", name: "Beta" },
    ];

    return (
      <List<CustomItem, "customId">
        data={data}
        idKey="customId"
        renderItemRow={(item) => <span>{item.name}</span>}
      />
    );
  },
};
