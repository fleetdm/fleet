import React from "react";

import { highlightMatches } from "../helpers";

const baseClass = "command-palette";

interface IHighlightedLabelProps {
  text: string;
  query: string;
}

/**
 * Wraps matched ranges of `query` within `text` in <mark> elements so the
 * user can see which part of a result matched their search. Renders a
 * Fragment — callers control the surrounding element/class.
 */
const HighlightedLabel = ({
  text,
  query,
}: IHighlightedLabelProps): JSX.Element => {
  return (
    <>
      {highlightMatches(text, query).map((seg, i) =>
        seg.matched ? (
          // Index keys are safe — segments are derived synchronously from
          // the same text + query each render, so order is stable.
          // eslint-disable-next-line react/no-array-index-key
          <mark key={i} className={`${baseClass}__item-label-match`}>
            {seg.text}
          </mark>
        ) : (
          // eslint-disable-next-line react/no-array-index-key
          <React.Fragment key={i}>{seg.text}</React.Fragment>
        )
      )}
    </>
  );
};

export default HighlightedLabel;
