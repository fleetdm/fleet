import React from "react";

import { render, screen } from "@testing-library/react";

import LabelsTable from "./LabelsTable";

describe("LabelsTable", () => {
  it("Renders empty state when only builtin labels are provided", () => {});
  it("Only renders custom labels when custom and builtin labels are provided", () => {});
  it("Includes edit and delete actions for global admins", () => {});
  it("Includes edit and delete actions for a team admin on a label they authored, but not on a label they did not", () => {});
});
