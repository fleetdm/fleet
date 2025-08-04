import React from "react";
import { render, screen } from "@testing-library/react";

import ScheduleBatchScriptModal from "./ScheduleBatchScriptModal";

describe("ScheduleBatchScriptModal", () => {
  it("shows the correct heading for linux/macos scripts", () => {
    render(<ScheduleBatchScriptModal />);
    const heading = screen.getByRole("heading", {
      name: /Schedule Batch Script Modal/i,
    });
    expect(heading).toBeInTheDocument();
  });

  it("shows the correct heading for windows", () => {
    render(<ScheduleBatchScriptModal />);
    const heading = screen.getByRole("heading", {
      name: /Schedule Batch Script Modal/i,
    });
    expect(heading).toBeInTheDocument();
  });

  it("does not show the scheduling UI if 'run now' is selected", () => {
    render(<ScheduleBatchScriptModal />);
    const heading = screen.getByRole("heading", {
      name: /Schedule Batch Script Modal/i,
    });
    expect(heading).toBeInTheDocument();
  });

  it("shows the scheduling UI if 'schedule for later' is selected", () => {
    render(<ScheduleBatchScriptModal />);
    const heading = screen.getByRole("heading", {
      name: /Schedule Batch Script Modal/i,
    });
    expect(heading).toBeInTheDocument();
  });
    
  describe("run now", () => {
  });
    
  describe("schedule for later", () => {
    
    it("requires a valid date", () => {
        render(<ScheduleBatchScriptModal />);
        const heading = screen.getByRole("heading", {
        name: /Schedule Batch Script Modal/i,
        });
        expect(heading).toBeInTheDocument();
    });
      
    it("requires a valid time", () => {
        render(<ScheduleBatchScriptModal />);
        const heading = screen.getByRole("heading", {
        name: /Schedule Batch Script Modal/i,
        });
        expect(heading).toBeInTheDocument();
    });
      
      
    
    
});
