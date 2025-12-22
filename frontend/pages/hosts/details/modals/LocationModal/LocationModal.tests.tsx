import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockHostGeolocation } from "__mocks__/hostMock";
import LocationModal from "./LocationModal";

// Mock current time for time stamp test
beforeAll(() => {
  jest.useFakeTimers();
  jest.setSystemTime(new Date("2022-05-08T10:00:00Z"));
});

describe("LocationModal", () => {
  console.log("tests todo");
});
