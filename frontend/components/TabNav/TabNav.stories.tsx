import React, { useState } from "react";
import { Meta, StoryObj } from "@storybook/react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import TabNav from "./TabNav";

const meta: Meta<typeof TabNav> = {
  component: TabNav,
  title: "Components/TabNav",
  parameters: {
    backgrounds: {
      default: "light",
      values: [
        {
          name: "light",
          value: "#ffffff",
        },
        {
          name: "dark",
          value: "#333333",
        },
      ],
    },
  },
};

export default meta;

type Story = StoryObj<typeof TabNav>;

export const Default: Story = {
  render: () => {
    const [selectedTabIndex, setSelectedTabIndex] = useState(0);

    const platformSubNav = [
      { name: "Tab 1", type: "type1" },
      { name: "Tab 2", type: "type2" },
      { name: "Tab 3", type: "type3" },
    ];

    const renderPanel = (type: string) => {
      switch (type) {
        case "type1":
          return <div>Content for Tab 1</div>;
        case "type2":
          return <div>Content for Tab 2</div>;
        case "type3":
          return <div>Content for Tab 3</div>;
        default:
          return null;
      }
    };

    return (
      <TabNav>
        <Tabs
          onSelect={(index) => setSelectedTabIndex(index)}
          selectedIndex={selectedTabIndex}
        >
          <TabList>
            {platformSubNav.map((navItem) => (
              <Tab key={navItem.name} data-text={navItem.name}>
                {navItem.name}
              </Tab>
            ))}
          </TabList>
          {platformSubNav.map((navItem) => (
            <TabPanel key={navItem.type}>
              <div>{renderPanel(navItem.type)}</div>
            </TabPanel>
          ))}
        </Tabs>
      </TabNav>
    );
  },
};
