import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import mockServer from "test/mock-server";
import { customDeviceSoftwareHandler } from "test/handlers/device-handler";
import { createMockDeviceSoftware } from "__mocks__/deviceUserMock";

import SelfService from "./SelfService";

describe("SelfService", () => {
  it("should render the self service items correctly", async () => {
    mockServer.use(
      customDeviceSoftwareHandler({
        software: [
          createMockDeviceSoftware({ name: "test1" }),
          createMockDeviceSoftware({ name: "test2" }),
          createMockDeviceSoftware({ name: "test3" }),
        ],
        count: 3,
      })
    );

    const render = createCustomRenderer({ withBackendMock: true });

    render(
      <SelfService
        contactUrl={"http://example.com"}
        deviceToken={"123-456"}
        isSoftwareEnabled
        pathname={"/test"}
        queryParams={{
          page: 1,
          query: "",
          order_key: "name",
          order_direction: "asc",
          per_page: 10,
          vulnerable: true,
        }}
        router={createMockRouter()}
      />
    );

    // waiting for the device software data to render
    await screen.findByText("test1");

    expect(true).toBe(true);
    expect(screen.getByText("test1")).toBeInTheDocument();
    expect(screen.getByText("test2")).toBeInTheDocument();
    expect(screen.getByText("test3")).toBeInTheDocument();
    expect(screen.getByText("3 items")).toBeInTheDocument();
    screen.debug();
  });

  it("should render the contact link text if contact url is provided", () => {
    mockServer.use(customDeviceSoftwareHandler());

    const render = createCustomRenderer({ withBackendMock: true });

    const expectedUrl = "http://example.com";

    render(
      <SelfService
        contactUrl={expectedUrl}
        deviceToken={"123-456"}
        isSoftwareEnabled
        pathname={"/test"}
        queryParams={{
          page: 1,
          query: "test",
          order_key: "name",
          order_direction: "asc",
          per_page: 10,
          vulnerable: true,
        }}
        router={createMockRouter()}
      />
    );

    expect(screen.getByText("reach out to IT")).toBeInTheDocument();
    expect(screen.getByText("reach out to IT").getAttribute("href")).toBe(
      expectedUrl
    );
  });
});
