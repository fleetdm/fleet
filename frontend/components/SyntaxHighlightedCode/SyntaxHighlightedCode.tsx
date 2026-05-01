import React from "react";

import { syntaxHighlight } from "utilities/helpers";

interface ISyntaxHighlightedCodeProps {
  json: unknown;
  className?: string;
}

/**
 * Renders a JSON object as syntax-highlighted HTML inside a <pre> tag.
 *
 * This component wraps the `syntaxHighlight` utility which generates safe HTML
 * from trusted JSON data (all HTML entities are escaped before span tags are
 * added for styling). The dangerouslySetInnerHTML usage is intentional and safe
 * because the content is never derived from user input — it is always
 * serialized from a controlled JavaScript object.
 */
const SyntaxHighlightedCode = ({
  json,
  className,
}: ISyntaxHighlightedCodeProps): JSX.Element => {
  return (
    <pre
      className={className}
      // eslint-disable-next-line react/no-danger
      dangerouslySetInnerHTML={{ __html: syntaxHighlight(json) }}
    />
  );
};

export default SyntaxHighlightedCode;
