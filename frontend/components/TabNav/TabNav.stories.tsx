import React, { useState } from "react";
import { Meta, StoryObj } from "@storybook/react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import TabText from "components/TabText";
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
      { name: <TabText>Basic tab</TabText>, type: "type1" },
      { name: <TabText>Basic tab 2</TabText>, type: "type2" },
      {
        name: <TabText>Disabled tab</TabText>,
        type: "type3",
        disabled: true,
      },
      { name: <TabText count={3}>Tab with count</TabText>, type: "type4" },
      {
        name: (
          <TabText count={20} isErrorCount>
            Tab with error count
          </TabText>
        ),
        type: "type5",
      },
    ];

    const renderPanel = (type: string) => {
      switch (type) {
        case "type1":
          return <div>Content for Tab 1</div>;
        case "type2":
          return <div>Content for Tab 2</div>;
        case "type3":
          return <div>Content for Tab 3</div>;
        case "type4":
          return <div>Content for Tab 4</div>;
        case "type5":
          return <div>Content for Tab 5</div>;
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
              <Tab disabled={navItem.disabled}>
                <TabText>{navItem.name}</TabText>
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
