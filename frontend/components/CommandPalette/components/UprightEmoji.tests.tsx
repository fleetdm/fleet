import React from "react";
import { render } from "@testing-library/react";

import UprightEmoji, { splitEmojiSegments } from "./UprightEmoji";

describe("splitEmojiSegments", () => {
  it("returns a single non-emoji segment for an emoji-free string", () => {
    expect(splitEmojiSegments("Workstations")).toEqual([
      { text: "Workstations", isEmoji: false },
    ]);
  });

  it("returns an empty segment for an empty string", () => {
    expect(splitEmojiSegments("")).toEqual([{ text: "", isEmoji: false }]);
  });

  it("splits an emoji prefix from the trailing text", () => {
    expect(splitEmojiSegments("💻 Workstations")).toEqual([
      { text: "💻", isEmoji: true },
      { text: " Workstations", isEmoji: false },
    ]);
  });

  it("coalesces adjacent emoji into one segment", () => {
    expect(splitEmojiSegments("📱🔐 Personal mobile devices")).toEqual([
      { text: "📱🔐", isEmoji: true },
      { text: " Personal mobile devices", isEmoji: false },
    ]);
  });

  it("handles emoji inside the string (not just at the start)", () => {
    expect(splitEmojiSegments("Team 🚀 alpha")).toEqual([
      { text: "Team ", isEmoji: false },
      { text: "🚀", isEmoji: true },
      { text: " alpha", isEmoji: false },
    ]);
  });

  it("handles a ZWJ-joined emoji as a single segment", () => {
    // Man technologist: 👨 + ZWJ + 💻 → one grapheme
    expect(splitEmojiSegments("👨‍💻 Engineering")).toEqual([
      { text: "👨‍💻", isEmoji: true },
      { text: " Engineering", isEmoji: false },
    ]);
  });

  it("handles emoji modifiers (skin tone) without splitting the grapheme", () => {
    expect(splitEmojiSegments("👍🏽 Team")).toEqual([
      { text: "👍🏽", isEmoji: true },
      { text: " Team", isEmoji: false },
    ]);
  });

  it("handles modifier + ZWJ sequences as a single segment", () => {
    // Man technologist with medium skin tone: 👨 + 🏽 + ZWJ + 💻
    expect(splitEmojiSegments("👨🏽‍💻 Engineering")).toEqual([
      { text: "👨🏽‍💻", isEmoji: true },
      { text: " Engineering", isEmoji: false },
    ]);
  });

  it("handles a regional indicator pair (flag emoji) as one segment", () => {
    // 🇺🇸 = U+1F1FA U+1F1F8
    expect(splitEmojiSegments("\u{1F1FA}\u{1F1F8} US fleet")).toEqual([
      { text: "\u{1F1FA}\u{1F1F8}", isEmoji: true },
      { text: " US fleet", isEmoji: false },
    ]);
  });

  it("handles emoji at the very end of the string", () => {
    expect(splitEmojiSegments("Sales team 🏆")).toEqual([
      { text: "Sales team ", isEmoji: false },
      { text: "🏆", isEmoji: true },
    ]);
  });
});

describe("UprightEmoji", () => {
  it("renders plain text when there are no emoji", () => {
    const { container } = render(<UprightEmoji text="Workstations" />);
    expect(container.textContent).toBe("Workstations");
    expect(container.querySelectorAll("span")).toHaveLength(0);
  });

  it("wraps emoji segments in a span with font-style: normal", () => {
    const { container } = render(<UprightEmoji text="💻 Workstations" />);
    const spans = container.querySelectorAll("span");
    expect(spans).toHaveLength(1);
    expect(spans[0].textContent).toBe("💻");
    expect(spans[0].style.fontStyle).toBe("normal");
    expect(container.textContent).toBe("💻 Workstations");
  });

  it("coalesces adjacent emoji into a single span", () => {
    const { container } = render(
      <UprightEmoji text="📱🔐 Personal mobile devices" />
    );
    const spans = container.querySelectorAll("span");
    expect(spans).toHaveLength(1);
    expect(spans[0].textContent).toBe("📱🔐");
  });

  it("preserves a ZWJ sequence as one upright span", () => {
    const { container } = render(<UprightEmoji text="👨‍💻 Engineering" />);
    const spans = container.querySelectorAll("span");
    expect(spans).toHaveLength(1);
    expect(spans[0].textContent).toBe("👨‍💻");
  });
});
