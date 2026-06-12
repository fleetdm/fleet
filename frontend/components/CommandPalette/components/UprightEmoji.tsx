import React from "react";

// Matches an emoji "grapheme" — a single emoji codepoint with optional
// variation selector / ZWJ continuations, OR a regional indicator pair
// (used for flag emoji like 🇺🇸). Browsers fake italic by shearing every
// glyph including emoji, so wrapping emoji in `font-style: normal` opts
// them out of the synthetic slant while leaving Latin text italic via
// the parent's `font-style: italic`.
//   \uFE0F = variation selector-16 (force emoji presentation)
//   \u200D = zero-width joiner (combines emoji into one grapheme,
//             e.g. 👨‍💻 → "man technologist")
const EMOJI_RUN_RE = /(?:\p{Extended_Pictographic}(?:\p{Emoji_Modifier})?\uFE0F?(?:\u200D\p{Extended_Pictographic}(?:\p{Emoji_Modifier})?\uFE0F?)*|\p{Regional_Indicator}\p{Regional_Indicator})/gu;

export interface IEmojiSegment {
  text: string;
  isEmoji: boolean;
}

/**
 * Split `text` into alternating emoji vs non-emoji segments. Pure helper
 * exposed for testing; production callers use the <UprightEmoji /> wrapper.
 *
 * Returns a single non-emoji segment when no emoji are present, so callers
 * can fast-path the bare-string case without diffing array shapes.
 */
export const splitEmojiSegments = (text: string): IEmojiSegment[] => {
  if (!text) return [{ text: "", isEmoji: false }];
  // Collect match positions via String.replace with a callback. We don't
  // actually replace anything — replace just exposes (match, offset) for
  // each hit. Using replace instead of String.matchAll keeps the helper
  // compatible with pre-ES2020 lib targets and sidesteps mutating the
  // global regex's lastIndex.
  const matches: Array<{ start: number; end: number }> = [];
  text.replace(EMOJI_RUN_RE, (match, offset) => {
    const start = offset as number;
    matches.push({ start, end: start + match.length });
    return match;
  });

  const segments: IEmojiSegment[] = [];
  let cursor = 0;
  // Coalesce adjacent emoji matches (e.g. "📱🔐") into one segment so the
  // DOM stays compact and the span attribute count doesn't balloon for
  // rows that prefix two or three emoji.
  let pendingEmojiStart = -1;
  let pendingEmojiEnd = -1;
  const flushEmoji = () => {
    if (pendingEmojiStart === -1) return;
    segments.push({
      text: text.slice(pendingEmojiStart, pendingEmojiEnd),
      isEmoji: true,
    });
    pendingEmojiStart = -1;
    pendingEmojiEnd = -1;
  };
  matches.forEach(({ start, end }) => {
    if (start === pendingEmojiEnd) {
      pendingEmojiEnd = end;
      cursor = end;
      return;
    }
    flushEmoji();
    if (start > cursor) {
      segments.push({ text: text.slice(cursor, start), isEmoji: false });
    }
    pendingEmojiStart = start;
    pendingEmojiEnd = end;
    cursor = end;
  });
  flushEmoji();
  if (cursor < text.length) {
    segments.push({ text: text.slice(cursor), isEmoji: false });
  }
  if (segments.length === 0) return [{ text, isEmoji: false }];
  return segments;
};

interface IUprightEmojiProps {
  text: string;
}

/**
 * Renders `text` so emoji glyphs stay upright even when the surrounding
 * element has `font-style: italic`. Use anywhere fleet names (or other
 * user-provided strings that may carry emoji) appear inside italicized
 * surfaces — synthetic italic shears emoji and looks broken.
 */
const UprightEmoji = ({ text }: IUprightEmojiProps): JSX.Element => {
  const segments = splitEmojiSegments(text);
  return (
    <>
      {segments.map((seg, i) =>
        seg.isEmoji ? (
          // Index keys are safe — segments derive synchronously from the
          // same input each render, so order is stable.
          // eslint-disable-next-line react/no-array-index-key
          <span key={i} style={{ fontStyle: "normal" }}>
            {seg.text}
          </span>
        ) : (
          // eslint-disable-next-line react/no-array-index-key
          <React.Fragment key={i}>{seg.text}</React.Fragment>
        )
      )}
    </>
  );
};

export default UprightEmoji;
