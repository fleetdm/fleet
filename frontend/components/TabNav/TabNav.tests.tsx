import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import TabText from "components/TabText";
import TabNav from "./TabNav";

describe("TabNav", () => {
  it("renders tabs and panels correctly", () => {
    render(
      <TabNav>
        <Tabs>
          <TabList>
            <Tab>
              <TabText>Tab 1</TabText>
            </Tab>
            <Tab>
              <TabText>Tab 2</TabText>
            </Tab>
          </TabList>
          <TabPanel>
            <div>Content for Tab 1</div>
          </TabPanel>
          <TabPanel>
            <div>Content for Tab 2</div>
          </TabPanel>
        </Tabs>
      </TabNav>
    );

    // Check if tabs are rendered
    expect(screen.getByText("Tab 1")).toBeInTheDocument();
    expect(screen.getByText("Tab 2")).toBeInTheDocument();

    // Check if the first panel content is rendered by default
    expect(screen.getByText("Content for Tab 1")).toBeInTheDocument();
    expect(screen.queryByText("Content for Tab 2")).not.toBeInTheDocument();
  });

  it("switches tabs and displays the correct panel content", () => {
    render(
      <TabNav>
        <Tabs>
          <TabList>
            <Tab>
              <TabText>Tab 1</TabText>
            </Tab>
            <Tab>
              <TabText>Tab 2</TabText>
            </Tab>
          </TabList>
          <TabPanel>
            <div>Content for Tab 1</div>
          </TabPanel>
          <TabPanel>
            <div>Content for Tab 2</div>
          </TabPanel>
        </Tabs>
      </TabNav>
    );

    // Switch to the second tab
    fireEvent.click(screen.getByText("Tab 2"));

    // Check if the second panel content is displayed
    expect(screen.getByText("Content for Tab 2")).toBeInTheDocument();
    expect(screen.queryByText("Content for Tab 1")).not.toBeInTheDocument();
  });
});
